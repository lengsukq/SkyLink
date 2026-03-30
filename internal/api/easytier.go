package api

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/easytier"
	"github.com/skylink/skylink/internal/store"
)

// easyTierHostMutationAbort 非 Windows 主机上返回 501 并中止处理。
func easyTierHostMutationAbort(c *gin.Context) bool {
	if easytier.EasyTierSupportedOnHost() {
		return false
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": easytier.HostNotSupportedMessage,
		"code":  "easytier_windows_only",
	})
	return true
}

// daemonStartRequest 启动/重启 daemon 时可选传入的版本，用于解析二进制路径。
type daemonStartRequest struct {
	ImageTag string `json:"image_tag"`
}

type easyTierProfileUpsertBody struct {
	ID     string                `json:"id"`
	Name   string                `json:"name"`
	Config *store.EasyTierConfig `json:"config"`
}

func getDaemonStartBody(c *gin.Context) daemonStartRequest {
	var body daemonStartRequest
	_ = c.ShouldBindJSON(&body)
	return body
}

func (s *Server) listEasyTierProfiles(c *gin.Context) {
	ps, err := s.store.GetEasyTierProfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ps)
}

func (s *Server) createEasyTierProfile(c *gin.Context) {
	var body easyTierProfileUpsertBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Config == nil {
		body.Config = &store.EasyTierConfig{}
	}
	profile, err := s.store.UpsertEasyTierProfile(store.EasyTierProfile{
		ID:     body.ID,
		Name:   body.Name,
		Config: *body.Config,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (s *Server) updateEasyTierProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("id"))
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile id is required"})
		return
	}
	var body easyTierProfileUpsertBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Config == nil {
		body.Config = &store.EasyTierConfig{}
	}
	profile, err := s.store.UpsertEasyTierProfile(store.EasyTierProfile{
		ID:     profileID,
		Name:   body.Name,
		Config: *body.Config,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (s *Server) deleteEasyTierProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("id"))
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile id is required"})
		return
	}
	if err := s.store.DeleteEasyTierProfile(profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), easyTierStartTimeout)
	defer cancel()
	_ = s.easyTierDaemons.Stop(ctx, profileID)
	c.JSON(http.StatusOK, gin.H{"message": "profile deleted"})
}

func (s *Server) setEasyTierActiveProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("id"))
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile id is required"})
		return
	}
	if err := s.store.SetEasyTierActiveProfile(profileID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active_profile_id": profileID})
}

// getEasyTierConfig 返回当前 EasyTier 配置（从 store 读取）
func (s *Server) getEasyTierConfig(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cfg := &profile.Config
	if cfg.ImageTag == "" {
		cfg.ImageTag = config.DefaultEasyTierTag
	}
	if cfg.RPCPortal == "" {
		cfg.RPCPortal = config.DefaultEasyTierRPC
	}
	cfg.EnvFilePath = s.resolveEasyTierEnvPath(profile.ID, cfg.EnvFilePath)
	c.JSON(http.StatusOK, cfg)
}

// putEasyTierConfig 保存 EasyTier 配置，并在需要时写入 env 文件供外部进程使用
func (s *Server) putEasyTierConfig(c *gin.Context) {
	if easyTierHostMutationAbort(c) {
		return
	}
	var cfg store.EasyTierConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	profile.Config = cfg
	if _, err := s.store.UpsertEasyTierProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	envPath := s.resolveEasyTierEnvPath(profile.ID, cfg.EnvFilePath)
	if envPath != "" {
		if err := s.store.WriteEasyTierEnv(envPath, &cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write env file: " + err.Error()})
			return
		}
	}

	// 若启用 daemon 模式，则在保存配置后尝试重启 EasyTier daemon
	if s.easyTierDaemons != nil && s.easyTierCfg != nil && s.easyTierCfg.DaemonEnabled && cfg.Enabled {
		go s.restartEasyTierDaemon(profile.ID, envPath, cfg.ImageTag)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "配置已保存。若已修改版本或网络参数，请重启或刷新 EasyTier 守护进程使配置生效。",
	})
}

// getEasyTierStatus 返回 mesh 状态（本机 IP、peers、routes、版本）
func (s *Server) getEasyTierStatus(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.respondEasyTierStatus(c, profile.Config)
}

// getEasyTierVersion 返回当前 EasyTier 运行版本
func (s *Server) getEasyTierVersion(c *gin.Context) {
	profile, _ := s.getCurrentEasyTierProfile()
	s.respondEasyTierVersion(c, profile.Config)
}

// getEasyTierVersionCheck 请求 GitHub 获取最新版本并比较
func (s *Server) getEasyTierVersionCheck(c *gin.Context) {
	latestTag, releaseURL, err := easytier.FetchLatestRelease()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"current_version":  "",
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
		"latest_version":    latestTag,
		"update_available":  updateAvailable,
		"release_url":       releaseURL,
		"release_notes_url": releaseURL,
	})
}

// postEasyTierDaemonStart 手动启动 EasyTier daemon（裸机场景）
func (s *Server) postEasyTierDaemonStart(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.startEasyTierDaemon(c, profile.ID, profile.Config)
}

func (s *Server) postEasyTierDaemonStartByProfile(c *gin.Context) {
	profile, err := s.getEasyTierProfileByID(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.startEasyTierDaemon(c, profile.ID, profile.Config)
}

func (s *Server) startEasyTierDaemon(c *gin.Context, profileID string, cfg store.EasyTierConfig) {
	if easyTierHostMutationAbort(c) {
		return
	}
	if s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier daemon mode is not enabled"})
		return
	}
	if !cfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier is not enabled in config"})
		return
	}
	envPath := s.resolveEasyTierEnvPath(profileID, cfg.EnvFilePath)
	if err := s.validateEasyTierProfileConflicts(profileID, cfg, envPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if envPath != "" {
		if err := s.store.WriteEasyTierEnv(envPath, &cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入 env 文件失败: " + err.Error()})
			return
		}
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), easyTierStartTimeout)
	defer cancel()
	if err := s.easyTierDaemons.Start(ctx, profileID, easytier.NewDaemonConfig(daemonPath, envPath)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "EasyTier daemon started"})
}

// postEasyTierDaemonStop 手动停止 EasyTier daemon
func (s *Server) postEasyTierDaemonStop(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.stopEasyTierDaemon(c, profile.ID)
}

func (s *Server) postEasyTierDaemonStopByProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("id"))
	s.stopEasyTierDaemon(c, profileID)
}

func (s *Server) stopEasyTierDaemon(c *gin.Context, profileID string) {
	if easyTierHostMutationAbort(c) {
		return
	}
	if s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier daemon mode is not enabled"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), easyTierStartTimeout)
	defer cancel()
	if err := s.easyTierDaemons.Stop(ctx, profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "EasyTier daemon stopped"})
}

// postEasyTierDaemonRestart 手动重启 EasyTier daemon（先停止再启动）
func (s *Server) postEasyTierDaemonRestart(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.restartEasyTierDaemonAPI(c, profile.ID, profile.Config)
}

func (s *Server) postEasyTierDaemonRestartByProfile(c *gin.Context) {
	profile, err := s.getEasyTierProfileByID(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.restartEasyTierDaemonAPI(c, profile.ID, profile.Config)
}

func (s *Server) restartEasyTierDaemonAPI(c *gin.Context, profileID string, cfg store.EasyTierConfig) {
	if easyTierHostMutationAbort(c) {
		return
	}
	if s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier daemon mode is not enabled"})
		return
	}
	if !cfg.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "EasyTier is not enabled in config"})
		return
	}
	envPath := s.resolveEasyTierEnvPath(profileID, cfg.EnvFilePath)
	if err := s.validateEasyTierProfileConflicts(profileID, cfg, envPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if envPath != "" {
		if err := s.store.WriteEasyTierEnv(envPath, &cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入 env 文件失败: " + err.Error()})
			return
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), easyTierRestartTimeout)
	defer cancel()
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
	if err := s.easyTierDaemons.Restart(ctx, profileID, easytier.NewDaemonConfig(daemonPath, envPath)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "restart: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "EasyTier daemon restarted"})
}

// getEasyTierDaemonStatus 返回 EasyTier daemon 当前运行状态
func (s *Server) getEasyTierDaemonStatus(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.respondEasyTierDaemonStatus(c, profile.ID)
}

func (s *Server) getEasyTierDaemonStatusByProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("id"))
	s.respondEasyTierDaemonStatus(c, profileID)
}

func (s *Server) respondEasyTierDaemonStatus(c *gin.Context, profileID string) {
	if !easytier.EasyTierSupportedOnHost() {
		daemonMode := s.easyTierCfg != nil && s.easyTierCfg.DaemonEnabled
		c.JSON(http.StatusOK, gin.H{
			"running":                 false,
			"daemon_mode_enabled":     daemonMode,
			"easytier_host_supported": false,
			"hint":                    easytier.HostNotSupportedHint,
		})
		return
	}
	if s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusOK, gin.H{
			"running":             false,
			"daemon_mode_enabled": false,
		})
		return
	}
	state := s.easyTierDaemons.Status(profileID)
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
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.respondEasyTierDaemonLogs(c, profile.ID)
}

func (s *Server) getEasyTierDaemonLogsByProfile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("id"))
	s.respondEasyTierDaemonLogs(c, profileID)
}

func (s *Server) respondEasyTierDaemonLogs(c *gin.Context, profileID string) {
	if !easytier.EasyTierSupportedOnHost() {
		c.JSON(http.StatusOK, gin.H{"logs": "", "hint": easytier.HostNotSupportedHint})
		return
	}
	if s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		c.JSON(http.StatusOK, gin.H{"logs": ""})
		return
	}
	logs := s.easyTierDaemons.Logs(profileID)
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// postEasyTierDaemonReleasePort 解除 EasyTier 相关端口占用（RPC 15888 + 默认 listeners 11010–11013，以及配置中的 rpc_portal 端口）
func (s *Server) postEasyTierDaemonReleasePort(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.releaseEasyTierPorts(c, profile.Config)
}

func (s *Server) postEasyTierDaemonReleasePortByProfile(c *gin.Context) {
	profile, err := s.getEasyTierProfileByID(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.releaseEasyTierPorts(c, profile.Config)
}

func (s *Server) releaseEasyTierPorts(c *gin.Context, cfg store.EasyTierConfig) {
	if easyTierHostMutationAbort(c) {
		return
	}
	ports, err := buildEasyTierReleasePortList(cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if q := strings.TrimSpace(c.Query("port")); q != "" {
		requestedPort := easytier.ParsePortFromAddress(q)
		if requestedPort == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid port query parameter"})
			return
		}
		allowed := false
		for _, p := range ports {
			if p == requestedPort {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         "port is not in allowed EasyTier release list",
				"allowed_ports": ports,
			})
			return
		}
		ports = []int{requestedPort}
	}
	killed, portsFreed, err := easytier.KillProcessOnPorts(ports)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if killed == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "EasyTier 相关端口暂无占用进程", "ports": ports})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已结束占用端口的进程", "killed": killed, "ports_freed": portsFreed})
}

func (s *Server) restartEasyTierDaemon(profileID, envPath, imageTag string) {
	if !easytier.EasyTierSupportedOnHost() || s.easyTierDaemons == nil || s.easyTierCfg == nil || !s.easyTierCfg.DaemonEnabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), easyTierRestartTimeout)
	defer cancel()
	daemonPath := s.resolveDaemonPath(ctx, imageTag)
	_ = s.easyTierDaemons.Restart(ctx, profileID, easytier.NewDaemonConfig(daemonPath, envPath))
}

func (s *Server) getEasyTierStatusByProfile(c *gin.Context) {
	profile, err := s.getEasyTierProfileByID(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.respondEasyTierStatus(c, profile.Config)
}

func (s *Server) getEasyTierVersionByProfile(c *gin.Context) {
	profile, err := s.getEasyTierProfileByID(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.respondEasyTierVersion(c, profile.Config)
}

func (s *Server) getEasyTierStatuses(c *gin.Context) {
	ps, err := s.store.GetEasyTierProfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ps.Profiles))
	if !easytier.EasyTierSupportedOnHost() {
		for _, p := range ps.Profiles {
			rpc := p.Config.RPCPortal
			if strings.TrimSpace(rpc) == "" {
				rpc = config.DefaultEasyTierRPC
			}
			out = append(out, gin.H{
				"id":         p.ID,
				"name":       p.Name,
				"rpc_portal": rpc,
				"ok":         false,
				"error":      "unsupported host",
				"hint":       easytier.HostNotSupportedHint,
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"profiles":                  out,
			"active_profile_id":         ps.ActiveProfileID,
			"easytier_host_supported":   false,
			"host_unsupported_hint":     easytier.HostNotSupportedHint,
		})
		return
	}
	ctx := c.Request.Context()
	for _, p := range ps.Profiles {
		rpc := p.Config.RPCPortal
		if strings.TrimSpace(rpc) == "" {
			rpc = config.DefaultEasyTierRPC
		}
		item := gin.H{
			"id":         p.ID,
			"name":       p.Name,
			"rpc_portal": rpc,
		}
		client := s.easyTierRPCClient(ctx, p.Config)
		status, err := client.Status()
		if err != nil {
			item["ok"] = false
			item["error"] = err.Error()
		} else {
			item["ok"] = true
			item["status"] = status
		}
		out = append(out, item)
	}
	c.JSON(http.StatusOK, gin.H{"profiles": out, "active_profile_id": ps.ActiveProfileID})
}

var easyTierCLIRawTargets = map[string]struct{}{
	"peer":    {},
	"route":   {},
	"node":    {},
	"version": {},
}

// getEasyTierCLIOutput 返回 easytier-cli 子命令原始输出（peer/route/node/version），便于对照官方文档排查在网状态。
func (s *Server) getEasyTierCLIOutput(c *gin.Context) {
	profile, err := s.getCurrentEasyTierProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.respondEasyTierCLIOutput(c, profile.Config)
}

func (s *Server) getEasyTierCLIOutputByProfile(c *gin.Context) {
	profile, err := s.getEasyTierProfileByID(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.respondEasyTierCLIOutput(c, profile.Config)
}

func (s *Server) respondEasyTierCLIOutput(c *gin.Context, cfg store.EasyTierConfig) {
	if !easytier.EasyTierSupportedOnHost() {
		c.JSON(http.StatusOK, gin.H{
			"ok":   false,
			"hint": easytier.HostNotSupportedHint,
		})
		return
	}
	target := strings.TrimSpace(c.Query("target"))
	if _, ok := easyTierCLIRawTargets[target]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target must be one of: peer, route, node, version"})
		return
	}
	client := s.easyTierRPCClient(c.Request.Context(), cfg)
	stdout, stderr, err := client.CLISubcommandRaw(target)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":     false,
			"target": target,
			"stdout": stdout,
			"stderr": stderr,
			"error":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"target": target,
		"stdout": stdout,
		"stderr": stderr,
	})
}

func (s *Server) respondEasyTierStatus(c *gin.Context, cfg store.EasyTierConfig) {
	if !easytier.EasyTierSupportedOnHost() {
		c.JSON(http.StatusOK, gin.H{
			"ok":     false,
			"error":  "unsupported host",
			"hint":   easytier.HostNotSupportedHint,
			"peers":  []interface{}{},
			"routes": []interface{}{},
		})
		return
	}
	client := s.easyTierRPCClient(c.Request.Context(), cfg)
	st, err := client.Status()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":     false,
			"error":  err.Error(),
			"hint":   "请确认 EasyTier 守护进程已运行、RPC 地址可达，且 easytier-cli 与 easytier-core 同目录（或由 RuntimeDownloader 下载完整包）或已在 PATH 中。",
			"peers":  []interface{}{},
			"routes": []interface{}{},
		})
		return
	}
	c.JSON(http.StatusOK, st)
}

func (s *Server) respondEasyTierVersion(c *gin.Context, cfg store.EasyTierConfig) {
	if !easytier.EasyTierSupportedOnHost() {
		c.JSON(http.StatusOK, gin.H{
			"version": "",
			"error":   "unsupported host",
			"hint":    easytier.HostNotSupportedHint,
		})
		return
	}
	client := s.easyTierRPCClient(c.Request.Context(), cfg)
	v, err := client.Version()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"version": "",
			"error":   err.Error(),
			"hint":    "请确认 EasyTier 守护进程已运行、RPC 地址可达，且 easytier-cli 与 easytier-core 同目录或已在 PATH 中。",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}

func (s *Server) getCurrentEasyTierProfile() (store.EasyTierProfile, error) {
	ps, err := s.store.GetEasyTierProfiles()
	if err != nil {
		return store.EasyTierProfile{}, err
	}
	for _, p := range ps.Profiles {
		if p.ID == ps.ActiveProfileID {
			return p, nil
		}
	}
	if len(ps.Profiles) > 0 {
		return ps.Profiles[0], nil
	}
	return store.EasyTierProfile{
		ID:     store.DefaultEasyTierProfileID,
		Name:   store.DefaultEasyTierProfileName,
		Config: store.EasyTierConfig{},
	}, nil
}

func (s *Server) getEasyTierProfileByID(profileID string) (store.EasyTierProfile, error) {
	ps, err := s.store.GetEasyTierProfiles()
	if err != nil {
		return store.EasyTierProfile{}, err
	}
	for _, p := range ps.Profiles {
		if p.ID == profileID {
			return p, nil
		}
	}
	return store.EasyTierProfile{}, fmt.Errorf("easytier profile %q not found", profileID)
}

func (s *Server) resolveEasyTierEnvPath(profileID, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	if s.easyTierEnvPath == "" {
		return ""
	}
	ext := filepath.Ext(s.easyTierEnvPath)
	base := strings.TrimSuffix(s.easyTierEnvPath, ext)
	if strings.TrimSpace(profileID) == "" || profileID == store.DefaultEasyTierProfileID {
		return s.easyTierEnvPath
	}
	return base + "." + profileID + ext
}

func (s *Server) validateEasyTierProfileConflicts(profileID string, cfg store.EasyTierConfig, envPath string) error {
	ps, err := s.store.GetEasyTierProfiles()
	if err != nil {
		return err
	}
	currentRPC := strings.TrimSpace(cfg.RPCPortal)
	if currentRPC == "" {
		currentRPC = config.DefaultEasyTierRPC
	}
	for _, p := range ps.Profiles {
		if p.ID == profileID {
			continue
		}
		otherRPC := strings.TrimSpace(p.Config.RPCPortal)
		if otherRPC == "" {
			otherRPC = config.DefaultEasyTierRPC
		}
		if currentRPC == otherRPC {
			return fmt.Errorf("rpc_portal conflict: profile %q already uses %s", p.Name, otherRPC)
		}
		otherEnvPath := s.resolveEasyTierEnvPath(p.ID, p.Config.EnvFilePath)
		if strings.TrimSpace(envPath) != "" && strings.EqualFold(filepath.Clean(otherEnvPath), filepath.Clean(envPath)) {
			return fmt.Errorf("env_file_path conflict: profile %q already uses %s", p.Name, otherEnvPath)
		}
	}
	return nil
}

// getEasyTierPlatform 返回后端运行平台信息（OS/Arch）
func (s *Server) getEasyTierPlatform(c *gin.Context) {
	p := easytier.CurrentPlatform()
	c.JSON(http.StatusOK, gin.H{
		"os":                      p.OS,
		"arch":                    p.Arch,
		"label":                   p.OS + "/" + p.Arch,
		"easytier_host_supported": easytier.EasyTierSupportedOnHost(),
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
		"platforms":               platforms,
		"current":                 gin.H{"os": current.OS, "arch": current.Arch, "label": currentLabel},
		"easytier_host_supported": easytier.EasyTierSupportedOnHost(),
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
	if easyTierHostMutationAbort(c) {
		return
	}
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
	if !easytier.IsSupportedRuntimePlatform(platform) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "仅支持为 Windows amd64 / Windows arm64 下载 EasyTier 运行时",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), easyTierInstallTimeout)
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
	if easyTierHostMutationAbort(c) {
		return
	}
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
	if !easytier.IsSupportedRuntimePlatform(platform) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持移除 Windows amd64 / arm64 运行时"})
		return
	}
	if err := s.easyTierRuntime.RemoveDaemon(version, platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}
