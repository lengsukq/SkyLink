package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/cloudflare"
	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/ddns"
	"github.com/skylink/skylink/internal/drive/indexer"
	"github.com/skylink/skylink/internal/easytier"
	"github.com/skylink/skylink/internal/proxy"
	"github.com/skylink/skylink/internal/security"
	"github.com/skylink/skylink/internal/service"
	"github.com/skylink/skylink/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// Server 管理 API + 静态资源
type Server struct {
	store    *store.Store
	proxy    *proxy.Proxy
	staticFS fs.FS // 可选，内嵌的 web/dist

	passwordHash        atomic.Value // string, persistent admin password hash from store
	startupPasswordHash atomic.Value // string, per-process startup password hash (memory only)
	constantPassword    string       // optional constant plaintext password from env

	// Cloudflare multi-account support
	cfClients          map[int64]*cloudflare.Client
	cfClientsMu        sync.Mutex
	currentCFAccountID atomic.Value // int64, 0 means unset

	ddnsUpdater *ddns.Updater

	// EasyTier：默认 env 文件路径（与 DB 同目录的 easytier.env）
	easyTierEnvPath string

	// EasyTier daemon lifecycle (裸机场景，可选启用)
	easyTierDaemons easytier.MultiDaemonManager
	easyTierCfg     *config.EasyTier

	// EasyTier runtime（自动下载 easytier-daemon）
	easyTierRuntime *easytier.RuntimeDownloader

	cfAccountService *service.CFAccountService

	driveJWTSecret string
	driveIndexer   *indexer.Manager
}

// New 创建 API 服务；staticFS 可为 nil（不提供 GUI 时）；easyTierEnvPath 为 EasyTier env 文件默认路径，空则仅从 store 读配置不写文件
func New(st *store.Store, pr *proxy.Proxy, cf *cloudflare.Client, staticFS fs.FS, easyTierEnvPath string, etDaemons easytier.MultiDaemonManager, etCfg *config.EasyTier, etRuntime *easytier.RuntimeDownloader) *Server {
	s := &Server{
		store:            st,
		proxy:            pr,
		staticFS:         staticFS,
		easyTierEnvPath:  easyTierEnvPath,
		easyTierDaemons:  etDaemons,
		easyTierCfg:      etCfg,
		easyTierRuntime:  etRuntime,
		cfAccountService: service.NewCFAccountService(st),
		driveIndexer:     indexer.NewManager(),
	}
	s.initDriveJWTSecret()
	hash, err := st.GetAdminPasswordHash()
	if err != nil {
		log.Printf("load admin password hash failed: %v", err)
	}
	s.passwordHash.Store(hash)

	// Optional constant admin password from environment.
	// This password is not stored in the database and always remains valid when configured.
	if v := os.Getenv("SKYLINK_ADMIN_PASSWORD"); v != "" {
		s.constantPassword = v
		log.Printf("SkyLink constant admin password is configured via SKYLINK_ADMIN_PASSWORD.")
	}

	// Generate a per-startup admin password that is only valid for the lifetime of this process.
	if pw, err := security.GeneratePassword(18); err != nil {
		log.Printf("generate startup admin password failed: %v", err)
	} else {
		if hashed, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost); err != nil {
			log.Printf("hash startup admin password failed: %v", err)
		} else {
			s.startupPasswordHash.Store(string(hashed))
			// 必须与哈希一并告知运维，否则校验链里有该密码但无人知晓明文。
			log.Printf("SkyLink startup admin password (valid until restart): %s", pw)
		}
	}

	// Try to initialize current CF account ID from settings if present.
	if v, err := st.GetSetting("cf.current_account_id"); err != nil {
		log.Printf("load current CF account id failed: %v", err)
	} else if v != "" {
		if id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && id > 0 {
			s.currentCFAccountID.Store(id)
		}
	}

	s.startDDNSUpdater()

	// 裸机 daemon 模式下，根据配置尝试自动拉起 EasyTier
	s.maybeStartEasyTierDaemon()
	return s
}

func (s *Server) initDriveJWTSecret() {
	const key = "drive.jwt_secret"
	if v, err := s.store.GetSetting(key); err == nil && strings.TrimSpace(v) != "" {
		s.driveJWTSecret = strings.TrimSpace(v)
		return
	}
	sec, err := security.GeneratePassword(48)
	if err != nil {
		// fallback: process-only secret (tokens invalid after restart)
		s.driveJWTSecret = "skylink-drive-secret"
		return
	}
	s.driveJWTSecret = sec
	_ = s.store.SetSetting(key, sec)
}

// Handler 返回 Gin Engine，供挂载到 admin 端口；若需鉴权则中间件校验 secret
func (s *Server) Handler() http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	s.registerPublicRoutes(r)
	r.Use(s.authMiddleware())
	s.registerProtectedRoutes(r)
	r.NoRoute(s.serveFrontend)
	return r
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 静态资源与前端路由不需要鉴权；只保护 /api/*
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		// 整段 /api/drive/* 不在这里做管理员校验；按路径由 requireAdmin 或 driveAuthMiddleware 处理。
		if strings.HasPrefix(c.Request.URL.Path, "/api/drive/") {
			c.Next()
			return
		}
		// WebDAV 文件路由采用独立 Basic Auth；mapping 配置接口仍走 Bearer 鉴权。
		if strings.HasPrefix(c.Request.URL.Path, "/api/webdav/") &&
			!strings.HasPrefix(c.Request.URL.Path, "/api/webdav/mappings") {
			c.Next()
			return
		}
		// 预检请求放行
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		// 仅允许登录接口匿名访问；修改密码仍需通过 Authorization 认证
		if c.Request.Method == http.MethodPost && c.FullPath() == "/api/auth/login" {
			c.Next()
			return
		}

		pw := bearerToken(c.GetHeader("Authorization"))
		if pw == "" || !s.verifyPassword(pw) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func bearerToken(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(v, "Bearer "))
	}
	return v
}

func (s *Server) verifyPassword(pw string) bool {
	if pw == "" {
		return false
	}

	// 1. Constant plaintext password from environment (if configured).
	if s.constantPassword != "" &&
		subtle.ConstantTimeCompare([]byte(pw), []byte(s.constantPassword)) == 1 {
		return true
	}

	// 2. Persistent password stored in the database.
	if hash, _ := s.passwordHash.Load().(string); hash != "" {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil {
			return true
		}
	}

	// 3. Per-startup in-memory password.
	if startupHash, _ := s.startupPasswordHash.Load().(string); startupHash != "" {
		if bcrypt.CompareHashAndPassword([]byte(startupHash), []byte(pw)) == nil {
			return true
		}
	}

	return false
}

func (s *Server) requireCF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := s.cfClientForCurrentAccount(); err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.Next()
	}
}

func (s *Server) reloadProxy() {
	m, err := s.store.HostToBackend()
	if err != nil {
		log.Printf("reload proxy routes failed: %v", err)
		return
	}
	s.proxy.SetRoutes(m)
}

// cfClientForCurrentAccount 根据当前激活的 CF 账号返回或创建对应的 Cloudflare 客户端
func (s *Server) cfClientForCurrentAccount() (*cloudflare.Client, error) {
	id := s.currentCFAccountID.Load()
	var accountID int64
	if id64, ok := id.(int64); ok {
		accountID = id64
	}
	if accountID == 0 {
		// 尝试从 settings 懒加载一次
		v, err := s.store.GetSetting("cf.current_account_id")
		if err != nil {
			return nil, err
		}
		if v == "" {
			return nil, errors.New("cloudflare account not configured")
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil || parsed <= 0 {
			return nil, errors.New("invalid cloudflare account id in settings")
		}
		accountID = parsed
		s.currentCFAccountID.Store(accountID)
	}

	s.cfClientsMu.Lock()
	defer s.cfClientsMu.Unlock()

	if s.cfClients == nil {
		s.cfClients = make(map[int64]*cloudflare.Client)
	}
	if client, ok := s.cfClients[accountID]; ok {
		return client, nil
	}

	acc, err := s.store.GetCFAccount(accountID)
	if err != nil {
		return nil, err
	}
	if acc == nil || strings.TrimSpace(acc.APIToken) == "" {
		return nil, errors.New("cloudflare account not found or token empty")
	}

	client := cloudflare.New(strings.TrimSpace(acc.APIToken))
	s.cfClients[accountID] = client
	return client, nil
}

// setCurrentCFAccountID 设置当前激活的 CF 账号 ID，并清理旧客户端缓存
func (s *Server) setCurrentCFAccountID(id int64) error {
	if id <= 0 {
		return errors.New("invalid cloudflare account id")
	}
	if err := s.store.SetSetting("cf.current_account_id", strconv.FormatInt(id, 10)); err != nil {
		return err
	}
	s.currentCFAccountID.Store(id)
	s.cfClientsMu.Lock()
	defer s.cfClientsMu.Unlock()
	s.cfClients = make(map[int64]*cloudflare.Client)
	return nil
}

// getCFClientByAccountID 根据 CF 账号 ID 返回或创建对应的 Cloudflare 客户端（供 DDNS 等按账号更新使用）
func (s *Server) getCFClientByAccountID(accountID int64) (*cloudflare.Client, error) {
	if accountID <= 0 {
		return nil, errors.New("invalid cloudflare account id")
	}
	s.cfClientsMu.Lock()
	defer s.cfClientsMu.Unlock()
	if s.cfClients == nil {
		s.cfClients = make(map[int64]*cloudflare.Client)
	}
	if client, ok := s.cfClients[accountID]; ok {
		return client, nil
	}
	acc, err := s.store.GetCFAccount(accountID)
	if err != nil {
		return nil, err
	}
	if acc == nil || strings.TrimSpace(acc.APIToken) == "" {
		return nil, errors.New("cloudflare account not found or token empty")
	}
	client := cloudflare.New(strings.TrimSpace(acc.APIToken))
	s.cfClients[accountID] = client
	return client, nil
}

func (s *Server) startDDNSUpdater() {
	s.ddnsUpdater = ddns.NewUpdater(
		func(item ddns.DDNSItem, newIP string) error {
			cfClient, err := s.getCFClientByAccountID(item.CFAccountID)
			if err != nil {
				log.Printf("ddns: skip config id=%d account=%d: %v", item.ID, item.CFAccountID, err)
				return err
			}
			recordType := item.RecordType
			if recordType != ddns.RecordTypeAAAA {
				recordType = ddns.RecordTypeA
			}
			_, err = cfClient.UpdateDNSRecord(item.ZoneID, item.RecordID, recordType, item.RecordName, newIP, cloudflare.TTLAuto, false)
			if err != nil {
				return err
			}
			return s.store.UpdateDDNSLastResult(item.ID, newIP)
		},
		func() ([]ddns.DDNSItem, error) {
			list, err := s.store.ListEnabledDDNSConfigs(0)
			if err != nil {
				return nil, err
			}
			out := make([]ddns.DDNSItem, len(list))
			for i := range list {
				rt := list[i].RecordType
				if rt != ddns.RecordTypeAAAA {
					rt = ddns.RecordTypeA
				}
				out[i] = ddns.DDNSItem{
					ID:          list[i].ID,
					CFAccountID: list[i].CFAccountID,
					ZoneID:      list[i].ZoneID,
					RecordName:  list[i].RecordName,
					RecordID:    list[i].RecordID,
					RecordType:  rt,
					IntervalMin: list[i].IntervalMin,
				}
			}
			return out, nil
		},
	)
	s.ddnsUpdater.Start()
}

// StopDDNS 停止 DDNS 后台更新（进程退出时由 main 调用）
func (s *Server) StopDDNS() {
	if s.ddnsUpdater != nil {
		s.ddnsUpdater.Stop()
	}
}

// StopEasyTierDaemon 在进程退出前停止 EasyTier daemon（若启用）
func (s *Server) StopEasyTierDaemon() {
	if s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()
	if err := s.easyTierDaemons.StopAll(ctx); err != nil {
		log.Printf("stop EasyTier daemon failed: %v", err)
	}
}

func (s *Server) maybeStartEasyTierDaemon() {
	if s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		return
	}
	if !easytier.EasyTierSupportedOnHost() {
		log.Printf("EasyTier autostart skipped: host OS is not Windows")
		return
	}
	autostart, err := s.store.GetEasyTierAutostart()
	if err != nil {
		log.Printf("load EasyTier autostart setting failed: %v", err)
		return
	}
	if !autostart {
		return
	}
	ps, err := s.store.GetEasyTierProfiles()
	if err != nil {
		log.Printf("load EasyTier profiles for daemon failed: %v", err)
		return
	}
	for _, profile := range ps.Profiles {
		if !profile.Config.Enabled {
			continue
		}
		envPath := s.resolveEasyTierEnvPath(profile.ID, profile.Config.EnvFilePath)
		daemonPath := s.resolveDaemonPath(context.Background(), profile.Config.ImageTag)
		ctx, cancel := context.WithTimeout(context.Background(), easyTierStartTimeout)
		err := s.easyTierDaemons.Start(ctx, profile.ID, easytier.NewDaemonConfig(daemonPath, envPath))
		cancel()
		if err != nil {
			log.Printf("auto start EasyTier daemon failed (%s): %v", profile.ID, err)
		}
	}
}

// resolveDaemonPath 根据配置和自动下载器决定要使用的 easytier-daemon 路径。
// 优先级：显式 DaemonPath（含路径时）> RuntimeDownloader 下载路径 > 默认二进制名（PATH）。
// 当 daemon_path 仅为 "easytier-core"（无目录）时视为未配置，优先使用已下载的运行时。
func (s *Server) resolveDaemonPath(ctx context.Context, imageTag string) string {
	if s.easyTierCfg != nil && isExplicitDaemonPath(s.easyTierCfg.DaemonPath) {
		return s.easyTierCfg.DaemonPath
	}
	if s.easyTierRuntime != nil {
		version := imageTag
		if strings.TrimSpace(version) == "" {
			version = config.DefaultEasyTierTag
		}
		if path, err := s.easyTierRuntime.EnsureDaemon(ctx, version, easytier.CurrentPlatform()); err == nil && strings.TrimSpace(path) != "" {
			return path
		}
	}
	return easytier.DefaultDaemonBinary()
}

// isExplicitDaemonPath 为 true 表示配置中显式指定了可执行文件路径（绝对路径或含目录），
// 而非仅默认二进制名 "easytier-core"（此时应优先使用 RuntimeDownloader 已下载的二进制）。
func isExplicitDaemonPath(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	if filepath.IsAbs(p) {
		return true
	}
	cleaned := filepath.Clean(p)
	return strings.Contains(cleaned, string(filepath.Separator))
}
