package api

import (
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/cloudflare"
	"github.com/skylink/skylink/internal/proxy"
	"github.com/skylink/skylink/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// Server 管理 API + 静态资源
type Server struct {
	store    *store.Store
	proxy    *proxy.Proxy
	cf       *cloudflare.Client
	staticFS fs.FS // 可选，内嵌的 web/dist

	passwordHash atomic.Value // string
}

// New 创建 API 服务；staticFS 可为 nil（不提供 GUI 时）
func New(st *store.Store, pr *proxy.Proxy, cf *cloudflare.Client, staticFS fs.FS) *Server {
	s := &Server{store: st, proxy: pr, cf: cf, staticFS: staticFS}
	hash, err := st.GetAdminPasswordHash()
	if err != nil {
		log.Printf("load admin password hash failed: %v", err)
	}
	s.passwordHash.Store(hash)
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

	// DDNS
	r.GET("/api/ddns", s.listDDNS)
	r.POST("/api/ddns", s.addDDNS)
	r.PUT("/api/ddns/:id", s.updateDDNS)
	r.DELETE("/api/ddns/:id", s.deleteDDNS)
	r.GET("/api/ddns/ip", s.getPublicIP)

	// 仪表盘汇总
	r.GET("/api/stats", s.stats)

	// 其它路径回退到内嵌静态前端（SPA）
	r.NoRoute(s.serveFrontend)
	return r
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 允许登录接口匿名访问
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
	hash, _ := s.passwordHash.Load().(string)
	if hash == "" {
		// 正常情况下启动时一定会生成；这里兜底拒绝
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func (s *Server) requireCF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.cf == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "cloudflare not configured"})
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
