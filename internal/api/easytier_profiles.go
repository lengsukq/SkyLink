package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skylink/skylink/internal/store"
)

type easyTierProfileUpsertBody struct {
	ID     string                `json:"id"`
	Name   string                `json:"name"`
	Config *store.EasyTierConfig `json:"config"`
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
