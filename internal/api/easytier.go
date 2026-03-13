package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/easytier"
	"github.com/skylink/skylink/internal/store"
)

// getEasyTierConfig 返回当前 EasyTier 配置（从 store 读取）
func (s *Server) getEasyTierConfig(c *gin.Context) {
	cfg, err := s.store.GetEasyTierConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		cfg = &store.EasyTierConfig{}
	}
	if cfg.ImageTag == "" {
		cfg.ImageTag = config.DefaultEasyTierTag
	}
	if cfg.RPCPortal == "" {
		cfg.RPCPortal = config.DefaultEasyTierRPC
	}
	if cfg.EnvFilePath == "" && s.easyTierEnvPath != "" {
		cfg.EnvFilePath = s.easyTierEnvPath
	}
	c.JSON(http.StatusOK, cfg)
}

// putEasyTierConfig 保存 EasyTier 配置并写入 env 文件
func (s *Server) putEasyTierConfig(c *gin.Context) {
	var cfg store.EasyTierConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.store.SetEasyTierConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	envPath := cfg.EnvFilePath
	if envPath == "" && s.easyTierEnvPath != "" {
		envPath = s.easyTierEnvPath
	}
	if envPath != "" {
		if err := s.store.WriteEasyTierEnv(envPath, &cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write env file: " + err.Error()})
			return
		}
	}

	// 若启用 daemon 模式，则在保存配置后尝试重启 EasyTier daemon
	if s.easyTierDaemon != nil && s.easyTierCfg != nil && s.easyTierCfg.DaemonEnabled && cfg.Enabled {
		go s.restartEasyTierDaemon(envPath)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "配置已保存。若已修改版本或网络参数，请重启 EasyTier 容器使配置生效。",
	})
}

// getEasyTierStatus 返回 mesh 状态（本机 IP、peers、routes、版本）
func (s *Server) getEasyTierStatus(c *gin.Context) {
	cfg, err := s.store.GetEasyTierConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rpc := config.DefaultEasyTierRPC
	if cfg != nil && cfg.RPCPortal != "" {
		rpc = cfg.RPCPortal
	}
	client := easytier.NewClient("", rpc)
	st, err := client.Status()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":    false,
			"error": err.Error(),
			"peers": []interface{}{},
			"routes": []interface{}{},
		})
		return
	}
	c.JSON(http.StatusOK, st)
}

// getEasyTierVersion 返回当前 EasyTier 运行版本
func (s *Server) getEasyTierVersion(c *gin.Context) {
	cfg, _ := s.store.GetEasyTierConfig()
	rpc := config.DefaultEasyTierRPC
	if cfg != nil && cfg.RPCPortal != "" {
		rpc = cfg.RPCPortal
	}
	client := easytier.NewClient("", rpc)
	v, err := client.Version()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"version": "", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}

// getEasyTierVersionCheck 请求 GitHub 获取最新版本并比较
func (s *Server) getEasyTierVersionCheck(c *gin.Context) {
	latestTag, releaseURL, err := easytier.FetchLatestRelease()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"current_version":   "",
			"latest_version":   "",
			"update_available": false,
			"error":            err.Error(),
		})
		return
	}
	cfg, _ := s.store.GetEasyTierConfig()
	configTag := config.DefaultEasyTierTag
	if cfg != nil && cfg.ImageTag != "" {
		configTag = cfg.ImageTag
	}
	// 规范化比较：去掉 v 前缀
	norm := func(s string) string { return strings.TrimPrefix(strings.TrimSpace(s), "v") }
	current := norm(configTag)
	latest := norm(latestTag)
	updateAvailable := current != "" && latest != "" && current != latest
	c.JSON(http.StatusOK, gin.H{
		"current_version":   configTag,
		"latest_version":   latestTag,
		"update_available": updateAvailable,
		"release_url":      releaseURL,
		"release_notes_url": releaseURL,
	})
}

// getEasyTierVPNPortal 返回 VPN Portal（WireGuard）客户端配置文本
func (s *Server) getEasyTierVPNPortal(c *gin.Context) {
	cfg, err := s.store.GetEasyTierConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"config": "",
			"error":  err.Error(),
		})
		return
	}
	rpc := config.DefaultEasyTierRPC
	if cfg != nil && cfg.RPCPortal != "" {
		rpc = cfg.RPCPortal
	}
	client := easytier.NewClient("", rpc)
	text, err := client.VPNPortalConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"config": "",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"config": text,
	})
}

// postEasyTierDaemonStart 手动启动 EasyTier daemon（裸机场景）
func (s *Server) postEasyTierDaemonStart(c *gin.Context) {
	if s.easyTierDaemon == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier daemon mode is not enabled"})
		return
	}
	cfg, err := s.store.GetEasyTierConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil || !cfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier is not enabled in config"})
		return
	}
	envPath := cfg.EnvFilePath
	if envPath == "" && s.easyTierEnvPath != "" {
		envPath = s.easyTierEnvPath
	}
	daemonPath := s.resolveDaemonPath(c.Request.Context(), cfg.ImageTag)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	if err := s.easyTierDaemon.Start(ctx, easytier.DaemonConfig{
		BinaryPath: daemonPath,
		EnvFile:    envPath,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "EasyTier daemon started"})
}

// postEasyTierDaemonStop 手动停止 EasyTier daemon
func (s *Server) postEasyTierDaemonStop(c *gin.Context) {
	if s.easyTierDaemon == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier daemon mode is not enabled"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	if err := s.easyTierDaemon.Stop(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "EasyTier daemon stopped"})
}

// getEasyTierDaemonStatus 返回 EasyTier daemon 当前运行状态
func (s *Server) getEasyTierDaemonStatus(c *gin.Context) {
	if s.easyTierDaemon == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusOK, gin.H{
			"running":             false,
			"daemon_mode_enabled": false,
		})
		return
	}
	state := s.easyTierDaemon.Status()
	c.JSON(http.StatusOK, gin.H{
		"running":             state.Running,
		"pid":                 state.PID,
		"last_start_error":    state.LastStartErr,
		"started_at":          state.StartedAt,
		"daemon_mode_enabled": true,
	})
}

func (s *Server) restartEasyTierDaemon(envPath string) {
	if s.easyTierDaemon == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.easyTierDaemon.Stop(ctx); err != nil {
		return
	}
	daemonPath := s.resolveDaemonPath(ctx, "")
	_ = s.easyTierDaemon.Start(ctx, easytier.DaemonConfig{
		BinaryPath: daemonPath,
		EnvFile:    envPath,
	})
}

// getEasyTierPlatform 返回后端运行平台信息（OS/Arch）
func (s *Server) getEasyTierPlatform(c *gin.Context) {
	p := easytier.CurrentPlatform()
	c.JSON(http.StatusOK, gin.H{
		"os":    p.OS,
		"arch":  p.Arch,
		"label": p.OS + "/" + p.Arch,
	})
}

// postEasyTierRuntimeInstall 为当前平台和配置版本下载/准备 easytier-daemon 运行时
func (s *Server) postEasyTierRuntimeInstall(c *gin.Context) {
	if s.easyTierRuntime == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime downloader is not configured"})
		return
	}
	var body struct {
		Version string `json:"version"`
	}
	_ = c.ShouldBindJSON(&body)

	cfg, _ := s.store.GetEasyTierConfig()
	version := strings.TrimSpace(body.Version)
	if version == "" {
		if cfg != nil && strings.TrimSpace(cfg.ImageTag) != "" {
			version = cfg.ImageTag
		} else {
			version = config.DefaultEasyTierTag
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	path, err := s.easyTierRuntime.EnsureDaemon(ctx, version, easytier.CurrentPlatform())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"installed":   false,
			"version":     version,
			"binary_path": "",
			"error":       err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"installed":   true,
		"version":     version,
		"binary_path": path,
	})
}
