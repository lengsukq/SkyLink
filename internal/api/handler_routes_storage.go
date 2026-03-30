package api

import "github.com/gin-gonic/gin"

func (s *Server) registerWebDAVRoutes(r *gin.Engine) {
	r.GET("/api/webdav/mappings", s.listWebDavMappings)
	r.POST("/api/webdav/mappings", s.addWebDavMapping)
	r.PUT("/api/webdav/mappings/:id", s.updateWebDavMapping)
	r.DELETE("/api/webdav/mappings/:id", s.deleteWebDavMapping)
	r.POST("/api/webdav/mappings/:id/start", s.startWebDavMapping)
	r.POST("/api/webdav/mappings/:id/stop", s.stopWebDavMapping)
	r.POST("/api/webdav/mappings/:id/restart", s.restartWebDavMapping)
	r.GET("/api/webdav/mappings/:id/health", s.healthWebDavMapping)
	r.Any("/api/webdav/:id/*davPath", s.serveWebDAVByID)
}

func (s *Server) registerSMBRoutes(r *gin.Engine) {
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
