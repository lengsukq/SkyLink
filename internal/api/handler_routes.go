package api

import "github.com/gin-gonic/gin"

func (s *Server) registerPublicRoutes(r *gin.Engine) {
	r.POST("/api/auth/login", s.login)
	r.POST("/api/auth/password", s.changePassword)
	r.POST("/api/drive/auth/login", s.driveLogin)
}

func (s *Server) registerProtectedRoutes(r *gin.Engine) {
	driveAdmin := r.Group("/api/drive/accounts", s.requireAdmin())
	driveAdmin.GET("", s.listDriveAccounts)
	driveAdmin.POST("", s.createDriveAccount)
	driveAdmin.PUT("/:id", s.updateDriveAccount)
	driveAdmin.DELETE("/:id", s.deleteDriveAccount)
	driveAdmin.POST("/:id/reset-password", s.resetDriveAccountPassword)
	driveAdmin.POST("/:id/recount-used", s.recountDriveAccountUsed)
	driveAdmin.POST("/:id/index/rebuild", s.driveIndexRebuild)
	driveAdmin.GET("/:id/index/status", s.driveIndexStatus)

	driveAPI := r.Group("/api/drive", s.driveAuthMiddleware())
	driveAPI.GET("/files", s.driveListFiles)
	driveAPI.GET("/entries", s.driveListEntries)
	driveAPI.GET("/preview-url", s.drivePreviewURL)
	driveAPI.POST("/folders", s.driveMkdir)
	driveAPI.POST("/rename", s.driveRename)
	driveAPI.DELETE("/files", s.driveDelete)
	driveAPI.POST("/upload", s.driveUpload)
	driveAPI.GET("/download", s.driveDownload)
	driveAPI.POST("/index/rebuild", s.driveUserIndexRebuild)
	driveAPI.GET("/index/status", s.driveUserIndexStatus)
	r.GET("/api/drive/preview", s.drivePreviewServe)

	s.registerMappingRoutes(r)
	s.registerCloudflareRoutes(r)

	r.GET("/api/ddns", s.listDDNS)
	r.POST("/api/ddns", s.addDDNS)
	r.PUT("/api/ddns/:id", s.updateDDNS)
	r.DELETE("/api/ddns/:id", s.deleteDDNS)
	r.GET("/api/ddns/ip", s.getPublicIP)

	r.GET("/api/stats", s.stats)
	r.GET("/api/system/volumes", s.getSystemVolumes)
	r.GET("/api/fs/browse", s.getFSBrowse)
	r.GET("/api/settings", s.getSettings)
	r.PUT("/api/settings", s.updateSettings)

	s.registerEasyTierRoutes(r)

	r.GET("/api/webdav/mappings", s.listWebDavMappings)
	r.POST("/api/webdav/mappings", s.addWebDavMapping)
	r.PUT("/api/webdav/mappings/:id", s.updateWebDavMapping)
	r.DELETE("/api/webdav/mappings/:id", s.deleteWebDavMapping)
	r.POST("/api/webdav/mappings/:id/start", s.startWebDavMapping)
	r.POST("/api/webdav/mappings/:id/stop", s.stopWebDavMapping)
	r.POST("/api/webdav/mappings/:id/restart", s.restartWebDavMapping)
	r.GET("/api/webdav/mappings/:id/health", s.healthWebDavMapping)
	r.Any("/api/webdav/:id/*davPath", s.serveWebDAVByID)

	r.GET("/api/smb/mappings", s.listSMBMappings)
	r.POST("/api/smb/mappings/sync-local", s.syncLocalSMBMappings)
	r.POST("/api/smb/mappings", s.addSMBMapping)
	r.PUT("/api/smb/mappings/:id", s.updateSMBMapping)
	r.DELETE("/api/smb/mappings/:id", s.deleteSMBMapping)
	r.POST("/api/smb/mappings/:id/start", s.startSMBMapping)
	r.POST("/api/smb/mappings/:id/stop", s.stopSMBMapping)
	r.POST("/api/smb/mappings/:id/restart", s.restartSMBMapping)
	r.GET("/api/smb/mappings/:id/health", s.healthSMBMapping)
}
