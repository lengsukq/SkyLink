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
	PublicIPTimeout   = 30 * time.Second
	TickInterval      = time.Minute
)

// PublicIPProviders 获取公网 IP 的接口
var PublicIPProviders = []string{
	"https://api.ipify.org",
	"https://ifconfig.me/ip",
	"https://icanhazip.com",
}

// GetPublicIP 从任一 provider 获取当前公网 IPv4
func GetPublicIP(ctx context.Context) (string, error) {
	for _, u := range PublicIPProviders {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := trimIP(string(b))
		if ip != "" {
			return ip, nil
		}
	}
	return "", fmt.Errorf("could not get public IP from any provider")
}

func trimIP(s string) string {
	b := make([]byte, 0, len(s))
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			b = append(b, byte(c))
		}
	}
	return string(b)
}

// DDNSItem 一条需要更新的 DDNS 配置（由调用方从 store 取）
type DDNSItem struct {
	ID          int64
	ZoneID      string
	RecordName  string
	RecordID    string
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
	// 当前策略：每 TickInterval 对所有启用项执行一次更新；IntervalMin 暂仅存储与展示
	// 若需按每条 IntervalMin 节流：可依据 store 中 updated_at 做到期判断。
	ctx, cancel := context.WithTimeout(context.Background(), PublicIPTimeout)
	ip, err := GetPublicIP(ctx)
	cancel()
	if err != nil || ip == "" {
		return
	}
	for _, item := range list {
		if item.IntervalMin <= 0 {
			item.IntervalMin = DefaultIntervalMin
		}
		_ = u.update(item, ip)
	}
}
