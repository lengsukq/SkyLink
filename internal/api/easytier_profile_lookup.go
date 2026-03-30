package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/store"
)

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

