package ddns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultIntervalMin = 10
	PublicIPTimeout    = 30 * time.Second
	TickInterval       = time.Minute
)

// IPv4 公网 IP 获取接口
var PublicIPv4Providers = []string{
	"https://api.ipify.org",
	"https://ifconfig.me/ip",
	"https://icanhazip.com",
}

// IPv6 公网 IP 获取接口
var PublicIPv6Providers = []string{
	"https://api64.ipify.org",
	"https://v6.ident.me",
	"https://v6.icanhazip.com",
}

// GetPublicIP 从任一 provider 获取当前公网 IPv4（兼容旧接口）
func GetPublicIP(ctx context.Context) (string, error) {
	return GetPublicIPv4(ctx)
}

// GetPublicIPv4 获取当前公网 IPv4
func GetPublicIPv4(ctx context.Context) (string, error) {
	for _, u := range PublicIPv4Providers {
		ip, err := fetchIP(ctx, u, trimIPv4)
		if err == nil && ip != "" {
			return ip, nil
		}
	}
	return "", fmt.Errorf("could not get public IPv4 from any provider")
}

// GetPublicIPv6 获取当前公网 IPv6（无 IPv6 时返回空字符串不报错）
func GetPublicIPv6(ctx context.Context) (string, error) {
	for _, u := range PublicIPv6Providers {
		ip, err := fetchIP(ctx, u, trimIPv6)
		if err == nil && ip != "" {
			return ip, nil
		}
	}
	return "", nil
}

func fetchIP(ctx context.Context, url string, trim func(string) string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return trim(string(b)), nil
}

func trimIPv4(s string) string {
	b := make([]byte, 0, len(s))
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			b = append(b, byte(c))
		}
	}
	return string(b)
}

func trimIPv6(s string) string {
	b := make([]byte, 0, len(s))
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == ':' {
			b = append(b, byte(c))
		}
	}
	return string(b)
}

// RecordTypeA / RecordTypeAAAA DNS 记录类型常量
const RecordTypeA = "A"
const RecordTypeAAAA = "AAAA"

// DDNSItem 一条需要更新的 DDNS 配置（由调用方从 store 取）
type DDNSItem struct {
	ID          int64
	CFAccountID int64 // 所属 Cloudflare 账号，用于按账号取 API 客户端
	ZoneID      string
	RecordName  string
	RecordID    string
	RecordType  string // "A" 或 "AAAA"
	IntervalMin int
}

// Updater 根据 store 中的 DDNS 配置，定时更新 CF A 记录
type Updater struct {
	mu     sync.Mutex
	stop   chan struct{}
	update func(item DDNSItem, newIP string) error
	list   func() ([]DDNSItem, error)
}

// NewUpdater 创建 DDNS 更新器；update 用于调用 CF 更新 A 记录并写回 store，list 返回已启用的配置列表
func NewUpdater(
	update func(item DDNSItem, newIP string) error,
	list func() ([]DDNSItem, error),
) *Updater {
	return &Updater{
		stop:   make(chan struct{}),
		update: update,
		list:   list,
	}
}

// Start 启动后台定时更新
func (u *Updater) Start() {
	u.mu.Lock()
	defer u.mu.Unlock()
	select {
	case <-u.stop:
		u.stop = make(chan struct{})
	default:
	}
	go u.loop()
}

// Stop 停止
func (u *Updater) Stop() {
	u.mu.Lock()
	ch := u.stop
	u.mu.Unlock()
	select {
	case <-ch:
		return
	default:
		close(ch)
	}
}

func (u *Updater) loop() {
	tick := time.NewTicker(TickInterval)
	defer tick.Stop()
	for {
		select {
		case <-u.stop:
			return
		case <-tick.C:
			u.runOnce()
		}
	}
}

func (u *Updater) runOnce() {
	list, err := u.list()
	if err != nil || len(list) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), PublicIPTimeout)
	ipv4, err4 := GetPublicIPv4(ctx)
	ipv6, err6 := GetPublicIPv6(ctx)
	cancel()
	// 若 IPv4 都拿不到则本轮不更新（IPv6 可选）
	if err4 != nil || ipv4 == "" {
		return
	}
	for _, item := range list {
		if item.IntervalMin <= 0 {
			item.IntervalMin = DefaultIntervalMin
		}
		recordType := item.RecordType
		if recordType != RecordTypeAAAA {
			recordType = RecordTypeA
		}
		var ip string
		if recordType == RecordTypeAAAA {
			ip = ipv6
			if err6 != nil || ip == "" {
				continue
			}
		} else {
			ip = ipv4
		}
		_ = u.update(item, ip)
	}
}
