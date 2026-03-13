package api

import (
	"context"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/easytier"
	"github.com/skylink/skylink/internal/store"
)

// daemonStartRequest 启动/重启 daemon 时可选传入的版本，用于解析二进制路径。
type daemonStartRequest struct {
	ImageTag string `json:"image_tag"`
}

func getDaemonStartBody(c *gin.Context) daemonStartRequest {
	var body daemonStartRequest
	_ = c.ShouldBindJSON(&body)
	return body
}

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

// putEasyTierConfig 保存 EasyTier 配置，并在需要时写入 env 文件供外部进程使用
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
		"message": "配置已保存。若已修改版本或网络参数，请重启或刷新 EasyTier 守护进程使配置生效。",
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
		// 将底层错误透出，同时给出更友好的排查提示
		c.JSON(http.StatusOK, gin.H{
			"ok":    false,
			"error": err.Error(),
			"hint":  "请确认 EasyTier 守护进程已运行，RPC 地址可达，并且 easytier-cli 已安装且在 SKYLINK 进程可见的 PATH 中。",
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
		c.JSON(http.StatusOK, gin.H{
			"version": "",
			"error":   err.Error(),
			"hint":    "请确认 EasyTier 守护进程已运行，RPC 地址可达，并且 easytier-cli 已安装且在 SKYLINK 进程可见的 PATH 中。",
		})
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
			"hint":   "请确认已在 EasyTier 中启用 VPN Portal，守护进程已运行，并且 easytier-cli 已安装且在 SKYLINK 进程可见的 PATH 中。",
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
	imageTag := cfg.ImageTag
	if body := getDaemonStartBody(c); body.ImageTag != "" {
		imageTag = body.ImageTag
	}
	daemonPath := s.resolveDaemonPath(c.Request.Context(), imageTag)
	if !filepath.IsAbs(daemonPath) {
		if _, err := exec.LookPath(daemonPath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "未找到 EasyTier 守护进程可执行文件",
				"hint":  "请在「版本与运行时」中选择版本与平台并点击「下载」，或配置 daemon_path 指向已安装的 easytier-core。",
			})
			return
		}
	}
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

// postEasyTierDaemonRestart 手动重启 EasyTier daemon（先停止再启动）
func (s *Server) postEasyTierDaemonRestart(c *gin.Context) {
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	if err := s.easyTierDaemon.Stop(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stop: " + err.Error()})
		return
	}
	imageTag := cfg.ImageTag
	if body := getDaemonStartBody(c); body.ImageTag != "" {
		imageTag = body.ImageTag
	}
	daemonPath := s.resolveDaemonPath(ctx, imageTag)
	if !filepath.IsAbs(daemonPath) {
		if _, err := exec.LookPath(daemonPath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "未找到 EasyTier 守护进程可执行文件",
				"hint":  "请在「版本与运行时」中选择版本与平台并点击「下载」，或配置 daemon_path 指向已安装的 easytier-core。",
			})
			return
		}
	}
	if err := s.easyTierDaemon.Start(ctx, easytier.DaemonConfig{
		BinaryPath: daemonPath,
		EnvFile:    envPath,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "start: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "EasyTier daemon restarted"})
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
	out := gin.H{
		"running":             state.Running,
		"pid":                 state.PID,
		"last_start_error":    state.LastStartErr,
		"started_at":          state.StartedAt,
		"daemon_mode_enabled": true,
	}
	if state.BinaryPath != "" && s.easyTierRuntime != nil {
		if v := s.easyTierRuntime.VersionFromBinaryPath(state.BinaryPath); v != "" {
			out["started_version"] = v
		}
	}
	c.JSON(http.StatusOK, out)
}

// getEasyTierDaemonLogs 返回守护进程最近 stdout/stderr 输出
func (s *Server) getEasyTierDaemonLogs(c *gin.Context) {
	if s.easyTierDaemon == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusOK, gin.H{"logs": ""})
		return
	}
	logs := s.easyTierDaemon.Logs()
	c.JSON(http.StatusOK, gin.H{"logs": logs})
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

// getEasyTierReleases 返回 GitHub EasyTier releases 列表，供版本下拉使用
func (s *Server) getEasyTierReleases(c *gin.Context) {
	list, err := easytier.FetchReleases(30)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"releases": []easytier.Release{}, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"releases": list})
}

// getEasyTierPlatforms 返回支持的平台列表及当前平台，供平台下拉与默认值使用
func (s *Server) getEasyTierPlatforms(c *gin.Context) {
	platforms := easytier.SupportedPlatformsWithLabels()
	current := easytier.CurrentPlatform()
	currentLabel := current.OS + "/" + current.Arch
	c.JSON(http.StatusOK, gin.H{
		"platforms": platforms,
		"current":   gin.H{"os": current.OS, "arch": current.Arch, "label": currentLabel},
	})
}

// getEasyTierRuntimeList 返回所有已下载的版本+平台列表，供版本管理使用
func (s *Server) getEasyTierRuntimeList(c *gin.Context) {
	if s.easyTierRuntime == nil {
		c.JSON(http.StatusOK, gin.H{"items": []easytier.InstalledRuntime{}})
		return
	}
	list, err := s.easyTierRuntime.ListInstalled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"items": nil, "error": err.Error()})
		return
	}
	if list == nil {
		list = []easytier.InstalledRuntime{}
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

// getEasyTierRuntimeInstalled 查询指定版本+平台是否已安装，返回 installed 与 path
func (s *Server) getEasyTierRuntimeInstalled(c *gin.Context) {
	version := strings.TrimSpace(c.Query("version"))
	osVal := strings.TrimSpace(c.Query("os"))
	arch := strings.TrimSpace(c.Query("arch"))
	if version == "" || osVal == "" || arch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version, os, arch required"})
		return
	}
	if s.easyTierRuntime == nil {
		c.JSON(http.StatusOK, gin.H{"installed": false, "path": ""})
		return
	}
	platform := easytier.Platform{OS: osVal, Arch: arch}
	installed := s.easyTierRuntime.HasDaemon(version, platform)
	path := ""
	if installed {
		path = s.easyTierRuntime.DaemonPath(version, platform)
	}
	c.JSON(http.StatusOK, gin.H{"installed": installed, "path": path})
}

// postEasyTierRuntimeInstall 为指定或当前平台下载 easytier-daemon 运行时；body 可选 os、arch
func (s *Server) postEasyTierRuntimeInstall(c *gin.Context) {
	if s.easyTierRuntime == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime downloader is not configured"})
		return
	}
	var body struct {
		Version string `json:"version"`
		OS      string `json:"os"`
		Arch    string `json:"arch"`
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

	platform := easytier.CurrentPlatform()
	if body.OS != "" && body.Arch != "" {
		platform = easytier.Platform{OS: strings.TrimSpace(body.OS), Arch: strings.TrimSpace(body.Arch)}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	path, err := s.easyTierRuntime.EnsureDaemon(ctx, version, platform)
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

// deleteEasyTierRuntime 移除已下载的指定版本+平台运行时
func (s *Server) deleteEasyTierRuntime(c *gin.Context) {
	if s.easyTierRuntime == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime downloader is not configured"})
		return
	}
	var body struct {
		Version string `json:"version"`
		OS      string `json:"os"`
		Arch    string `json:"arch"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	version := strings.TrimSpace(body.Version)
	osVal := strings.TrimSpace(body.OS)
	arch := strings.TrimSpace(body.Arch)
	if version == "" || osVal == "" || arch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version, os, arch required"})
		return
	}
	platform := easytier.Platform{OS: osVal, Arch: arch}
	if err := s.easyTierRuntime.RemoveDaemon(version, platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}
