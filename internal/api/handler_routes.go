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
	s.registerWebDAVRoutes(r)
	s.registerSMBRoutes(r)
}
