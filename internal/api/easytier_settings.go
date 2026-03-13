package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// getEasyTierSettings 返回 EasyTier 级别的附加设置（目前仅自动启动开关）
func (s *Server) getEasyTierSettings(c *gin.Context) {
	autostart, err := s.store.GetEasyTierAutostart()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"autostart_on_startup": autostart,
	})
}

// updateEasyTierSettings 更新 EasyTier 级别设置；当开启自动启动时会严格校验配置是否完整
func (s *Server) updateEasyTierSettings(c *gin.Context) {
	var req struct {
		AutostartOnStartup *bool `json:"autostart_on_startup"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AutostartOnStartup == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	current, err := s.store.GetEasyTierAutostart()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	target := *req.AutostartOnStartup

	// 仅在从关闭切换为开启时做严格校验
	if !current && target {
		cfg, err := s.store.GetEasyTierConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cfg == nil || !cfg.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "EasyTier 未在配置中启用，请先在 EasyTier 页完成网络配置并开启“启用”。",
			})
			return
		}
		if strings.TrimSpace(cfg.NetworkName) == "" || strings.TrimSpace(cfg.NetworkSecret) == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "EasyTier 网络名或网络密钥未配置完整，请先在 EasyTier 页填写网络名与网络密钥。",
			})
			return
		}
		// 为避免误判独立模式，这里采取保守策略：要求 peers 非空，等价于“非独立模式下 peers 必须配置好”
		if strings.TrimSpace(cfg.Peers) == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "EasyTier 初始节点（peers）未配置，请先在 EasyTier 页填写至少一个初始节点。",
			})
			return
		}
	}

	if err := s.store.SetEasyTierAutostart(target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"autostart_on_startup": target,
	})
}

