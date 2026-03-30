package api

import "github.com/gin-gonic/gin"

func (s *Server) registerMappingRoutes(r *gin.Engine) {
	r.GET("/api/mappings", s.listMappings)
	r.POST("/api/mappings", s.addMapping)
	r.PUT("/api/mappings/:id", s.updateMapping)
	r.DELETE("/api/mappings/:id", s.deleteMapping)
	r.POST("/api/mappings/one-click", s.oneClickMapping)
}

func (s *Server) registerCloudflareRoutes(r *gin.Engine) {
	cfGroup := r.Group("/api/cf", s.requireCF())
	cfGroup.GET("/zones", s.listZones)
	cfGroup.GET("/zones/:zoneId/records", s.listDNSRecords)
	cfGroup.POST("/zones/:zoneId/records", s.createDNSRecord)
	cfGroup.PUT("/zones/:zoneId/records/:recordId", s.updateDNSRecord)
	cfGroup.DELETE("/zones/:zoneId/records/:recordId", s.deleteDNSRecord)

	cfAccountGroup := r.Group("/api/cf/accounts")
	cfAccountGroup.GET("", s.listCFAccounts)
	cfAccountGroup.POST("", s.createCFAccount)
	cfAccountGroup.POST("/validate", s.validateCFAccount)
	cfAccountGroup.PUT("/:id", s.updateCFAccount)
	cfAccountGroup.DELETE("/:id", s.deleteCFAccount)
	cfAccountGroup.PUT("/:id/activate", s.activateCFAccount)
}

func (s *Server) registerEasyTierRoutes(r *gin.Engine) {
	r.GET("/api/easytier/config", s.getEasyTierConfig)
	r.PUT("/api/easytier/config", s.putEasyTierConfig)
	r.GET("/api/easytier/profiles", s.listEasyTierProfiles)
	r.POST("/api/easytier/profiles", s.createEasyTierProfile)
	r.PUT("/api/easytier/profiles/:id", s.updateEasyTierProfile)
	r.DELETE("/api/easytier/profiles/:id", s.deleteEasyTierProfile)
	r.PUT("/api/easytier/profiles/active/:id", s.setEasyTierActiveProfile)
	r.GET("/api/easytier/profiles/:id/status", s.getEasyTierStatusByProfile)
	r.GET("/api/easytier/profiles/:id/version", s.getEasyTierVersionByProfile)
	r.POST("/api/easytier/profiles/:id/daemon/start", s.postEasyTierDaemonStartByProfile)
	r.POST("/api/easytier/profiles/:id/daemon/stop", s.postEasyTierDaemonStopByProfile)
	r.POST("/api/easytier/profiles/:id/daemon/restart", s.postEasyTierDaemonRestartByProfile)
	r.GET("/api/easytier/profiles/:id/daemon/status", s.getEasyTierDaemonStatusByProfile)
	r.GET("/api/easytier/profiles/:id/daemon/logs", s.getEasyTierDaemonLogsByProfile)
	r.GET("/api/easytier/profiles/:id/cli-output", s.getEasyTierCLIOutputByProfile)
	r.POST("/api/easytier/profiles/:id/daemon/release-port", s.postEasyTierDaemonReleasePortByProfile)
	r.GET("/api/easytier/status/all", s.getEasyTierStatuses)
	r.GET("/api/easytier/settings", s.getEasyTierSettings)
	r.PUT("/api/easytier/settings", s.updateEasyTierSettings)
	r.GET("/api/easytier/status", s.getEasyTierStatus)
	r.GET("/api/easytier/cli-output", s.getEasyTierCLIOutput)
	r.GET("/api/easytier/version", s.getEasyTierVersion)
	r.GET("/api/easytier/version/check", s.getEasyTierVersionCheck)
	r.GET("/api/easytier/platform", s.getEasyTierPlatform)
	r.GET("/api/easytier/releases", s.getEasyTierReleases)
	r.GET("/api/easytier/platforms", s.getEasyTierPlatforms)
	r.GET("/api/easytier/runtime/installed", s.getEasyTierRuntimeInstalled)
	r.GET("/api/easytier/runtime/list", s.getEasyTierRuntimeList)
	r.POST("/api/easytier/runtime/install", s.postEasyTierRuntimeInstall)
	r.DELETE("/api/easytier/runtime", s.deleteEasyTierRuntime)
	r.POST("/api/easytier/daemon/start", s.postEasyTierDaemonStart)
	r.POST("/api/easytier/daemon/stop", s.postEasyTierDaemonStop)
	r.POST("/api/easytier/daemon/restart", s.postEasyTierDaemonRestart)
	r.POST("/api/easytier/daemon/release-port", s.postEasyTierDaemonReleasePort)
	r.GET("/api/easytier/daemon/status", s.getEasyTierDaemonStatus)
	r.GET("/api/easytier/daemon/logs", s.getEasyTierDaemonLogs)
}
