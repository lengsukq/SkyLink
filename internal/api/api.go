package api

import (
	"crypto/subtle"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/cloudflare"
	"github.com/skylink/skylink/internal/proxy"
	"github.com/skylink/skylink/internal/security"
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
}

// New 创建 API 服务；staticFS 可为 nil（不提供 GUI 时）
func New(st *store.Store, pr *proxy.Proxy, cf *cloudflare.Client, staticFS fs.FS) *Server {
	s := &Server{store: st, proxy: pr, staticFS: staticFS}
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

	return s
}

// Handler 返回 Gin Engine，供挂载到 admin 端口；若需鉴权则中间件校验 secret
func (s *Server) Handler() http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())

	// 登录相关接口放在鉴权前
	r.POST("/api/auth/login", s.login)
	r.POST("/api/auth/password", s.changePassword)

	r.Use(s.authMiddleware())

	// 映射
	r.GET("/api/mappings", s.listMappings)
	r.POST("/api/mappings", s.addMapping)
	r.PUT("/api/mappings/:id", s.updateMapping)
	r.DELETE("/api/mappings/:id", s.deleteMapping)

	// 一键映射：同时写 CF CNAME + 本地反代
	r.POST("/api/mappings/one-click", s.oneClickMapping)

	// Cloudflare
	cfGroup := r.Group("/api/cf", s.requireCF())
	cfGroup.GET("/zones", s.listZones)
	cfGroup.GET("/zones/:zoneId/records", s.listDNSRecords)
	cfGroup.POST("/zones/:zoneId/records", s.createDNSRecord)
	cfGroup.PUT("/zones/:zoneId/records/:recordId", s.updateDNSRecord)
	cfGroup.DELETE("/zones/:zoneId/records/:recordId", s.deleteDNSRecord)

	// Cloudflare accounts (仍走密码鉴权，但不强制已选择账号)
	cfAccountGroup := r.Group("/api/cf/accounts")
	cfAccountGroup.GET("", s.listCFAccounts)
	cfAccountGroup.POST("", s.createCFAccount)
	cfAccountGroup.POST("/validate", s.validateCFAccount)
	cfAccountGroup.PUT("/:id", s.updateCFAccount)
	cfAccountGroup.DELETE("/:id", s.deleteCFAccount)
	cfAccountGroup.PUT("/:id/activate", s.activateCFAccount)

	// DDNS
	r.GET("/api/ddns", s.listDDNS)
	r.POST("/api/ddns", s.addDDNS)
	r.PUT("/api/ddns/:id", s.updateDDNS)
	r.DELETE("/api/ddns/:id", s.deleteDDNS)
	r.GET("/api/ddns/ip", s.getPublicIP)

	// 仪表盘汇总
	r.GET("/api/stats", s.stats)

	// 全局设置
	r.GET("/api/settings", s.getSettings)
	r.PUT("/api/settings", s.updateSettings)

	// 其它路径回退到内嵌静态前端（SPA）
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
