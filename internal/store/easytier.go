package store

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// EasyTier 配置在 settings 表中的 key 前缀
const (
	KeyETNetworkName   = "et.network_name"
	KeyETNetworkSecret = "et.network_secret"
	KeyETPeers         = "et.peers"
	KeyETIPv4          = "et.ipv4"
	KeyETEnabled       = "et.enabled"
	KeyETImageTag      = "et.image_tag"
	KeyETRPCPortal     = "et.rpc_portal"
	KeyETEnvFilePath   = "et.env_file_path"
	KeyETHostname      = "et.hostname"
	KeyETExternalNode  = "et.external_node"
	KeyETProxyNetworks = "et.proxy_networks"
	KeyETDHCP          = "et.dhcp"
	KeyETAutostart     = "et.autostart_on_startup"
	KeyETProfiles      = "et.profiles"
	KeyETActiveProfile = "et.active_profile_id"
)

const (
	DefaultEasyTierProfileID   = "default"
	DefaultEasyTierProfileName = "默认配置"
)

// GetEasyTierConfig 从 settings 表读取 EasyTier 配置
func (s *Store) GetEasyTierConfig() (*EasyTierConfig, error) {
	ps, err := s.GetEasyTierProfiles()
	if err != nil {
		return nil, err
	}
	profile, ok := findProfile(ps.Profiles, ps.ActiveProfileID)
	if !ok {
		return &EasyTierConfig{}, nil
	}
	cfg := profile.Config
	return &cfg, nil
}

// SetEasyTierConfig 将 EasyTier 配置写入 settings 表
func (s *Store) SetEasyTierConfig(c *EasyTierConfig) error {
	if c == nil {
		return nil
	}
	ps, err := s.GetEasyTierProfiles()
	if err != nil {
		return err
	}
	if len(ps.Profiles) == 0 {
		ps.Profiles = []EasyTierProfile{{
			ID:     DefaultEasyTierProfileID,
			Name:   DefaultEasyTierProfileName,
			Config: EasyTierConfig{},
		}}
	}
	if strings.TrimSpace(ps.ActiveProfileID) == "" {
		ps.ActiveProfileID = DefaultEasyTierProfileID
	}
	i, ok := findProfileIndex(ps.Profiles, ps.ActiveProfileID)
	if !ok {
		ps.Profiles = append(ps.Profiles, EasyTierProfile{
			ID:     ps.ActiveProfileID,
			Name:   DefaultEasyTierProfileName,
			Config: EasyTierConfig{},
		})
		i = len(ps.Profiles) - 1
	}
	nextCfg := *c
	normalizeEasyTierConfig(&nextCfg)
	ps.Profiles[i].Config = nextCfg
	return s.SetEasyTierProfiles(ps)
}

func (s *Store) GetEasyTierProfiles() (*EasyTierProfiles, error) {
	raw, err := s.GetSetting(KeyETProfiles)
	if err != nil {
		return nil, err
	}
	active, err := s.GetSetting(KeyETActiveProfile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return s.migrateLegacyEasyTierProfiles(active)
	}
	var profiles []EasyTierProfile
	if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
		return s.migrateLegacyEasyTierProfiles(active)
	}
	normalized := normalizeProfiles(profiles)
	if strings.TrimSpace(active) == "" || !hasProfileID(normalized, active) {
		active = pickActiveProfileID(normalized)
	}
	ps := &EasyTierProfiles{
		Profiles:        normalized,
		ActiveProfileID: active,
	}
	return ps, nil
}

func (s *Store) SetEasyTierProfiles(ps *EasyTierProfiles) error {
	if ps == nil {
		return nil
	}
	normalized := normalizeProfiles(ps.Profiles)
	if len(normalized) == 0 {
		normalized = []EasyTierProfile{{
			ID:     DefaultEasyTierProfileID,
			Name:   DefaultEasyTierProfileName,
			Config: EasyTierConfig{},
		}}
	}
	activeID := strings.TrimSpace(ps.ActiveProfileID)
	if activeID == "" || !hasProfileID(normalized, activeID) {
		activeID = pickActiveProfileID(normalized)
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	if err := s.SetSetting(KeyETProfiles, string(payload)); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETActiveProfile, activeID); err != nil {
		return err
	}
	legacy, ok := findProfile(normalized, activeID)
	if !ok {
		return nil
	}
	return s.syncLegacyEasyTierKeys(legacy.Config)
}

func (s *Store) UpsertEasyTierProfile(profile EasyTierProfile) (*EasyTierProfile, error) {
	ps, err := s.GetEasyTierProfiles()
	if err != nil {
		return nil, err
	}
	p := profile
	if strings.TrimSpace(p.ID) == "" {
		p.ID = generateProfileID(p.Name, ps.Profiles)
	}
	if strings.TrimSpace(p.Name) == "" {
		p.Name = p.ID
	}
	normalizeEasyTierConfig(&p.Config)
	if i, ok := findProfileIndex(ps.Profiles, p.ID); ok {
		ps.Profiles[i] = p
	} else {
		ps.Profiles = append(ps.Profiles, p)
	}
	if strings.TrimSpace(ps.ActiveProfileID) == "" {
		ps.ActiveProfileID = p.ID
	}
	if err := s.SetEasyTierProfiles(ps); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) DeleteEasyTierProfile(profileID string) error {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil
	}
	ps, err := s.GetEasyTierProfiles()
	if err != nil {
		return err
	}
	next := make([]EasyTierProfile, 0, len(ps.Profiles))
	for _, p := range ps.Profiles {
		if p.ID != profileID {
			next = append(next, p)
		}
	}
	ps.Profiles = next
	if ps.ActiveProfileID == profileID {
		ps.ActiveProfileID = pickActiveProfileID(ps.Profiles)
	}
	return s.SetEasyTierProfiles(ps)
}

func (s *Store) SetEasyTierActiveProfile(profileID string) error {
	ps, err := s.GetEasyTierProfiles()
	if err != nil {
		return err
	}
	if !hasProfileID(ps.Profiles, profileID) {
		return fmt.Errorf("easytier profile %q not found", profileID)
	}
	ps.ActiveProfileID = profileID
	return s.SetEasyTierProfiles(ps)
}

// GetEasyTierAutostart 读取 EasyTier 是否在 SkyLink 启动时自动启动的开关（默认关闭）
func (s *Store) GetEasyTierAutostart() (bool, error) {
	v, err := s.GetSetting(KeyETAutostart)
	if err != nil {
		return false, err
	}
	if v == "" {
		return false, nil
	}
	return v == "1" || v == "true", nil
}

// SetEasyTierAutostart 写入 EasyTier 自动启动开关
func (s *Store) SetEasyTierAutostart(enabled bool) error {
	value := "0"
	if enabled {
		value = "1"
	}
	return s.SetSetting(KeyETAutostart, value)
}

func (s *Store) migrateLegacyEasyTierProfiles(active string) (*EasyTierProfiles, error) {
	cfg, err := s.loadLegacyEasyTierConfig()
	if err != nil {
		return nil, err
	}
	defaultProfile := EasyTierProfile{
		ID:     DefaultEasyTierProfileID,
		Name:   DefaultEasyTierProfileName,
		Config: cfg,
	}
	activeID := strings.TrimSpace(active)
	if activeID == "" {
		activeID = defaultProfile.ID
	}
	ps := &EasyTierProfiles{
		Profiles:        []EasyTierProfile{defaultProfile},
		ActiveProfileID: activeID,
	}
	if err := s.SetEasyTierProfiles(ps); err != nil {
		return nil, err
	}
	return ps, nil
}

func (s *Store) loadLegacyEasyTierConfig() (EasyTierConfig, error) {
	c := EasyTierConfig{}
	var err error
	if c.NetworkName, err = s.GetSetting(KeyETNetworkName); err != nil {
		return EasyTierConfig{}, err
	}
	if c.NetworkSecret, err = s.GetSetting(KeyETNetworkSecret); err != nil {
		return EasyTierConfig{}, err
	}
	if c.Peers, err = s.GetSetting(KeyETPeers); err != nil {
		return EasyTierConfig{}, err
	}
	if c.IPv4, err = s.GetSetting(KeyETIPv4); err != nil {
		return EasyTierConfig{}, err
	}
	if v, err := s.GetSetting(KeyETEnabled); err != nil {
		return EasyTierConfig{}, err
	} else {
		c.Enabled = v == "1" || v == "true"
	}
	if c.ImageTag, err = s.GetSetting(KeyETImageTag); err != nil {
		return EasyTierConfig{}, err
	}
	if c.RPCPortal, err = s.GetSetting(KeyETRPCPortal); err != nil {
		return EasyTierConfig{}, err
	}
	if c.EnvFilePath, err = s.GetSetting(KeyETEnvFilePath); err != nil {
		return EasyTierConfig{}, err
	}
	if c.Hostname, err = s.GetSetting(KeyETHostname); err != nil {
		return EasyTierConfig{}, err
	}
	if c.ExternalNode, err = s.GetSetting(KeyETExternalNode); err != nil {
		return EasyTierConfig{}, err
	}
	if c.ProxyNetworks, err = s.GetSetting(KeyETProxyNetworks); err != nil {
		return EasyTierConfig{}, err
	}
	if v, err := s.GetSetting(KeyETDHCP); err != nil {
		return EasyTierConfig{}, err
	} else {
		c.DHCP = v == "1" || v == "true"
	}
	normalizeEasyTierConfig(&c)
	return c, nil
}

func (s *Store) syncLegacyEasyTierKeys(c EasyTierConfig) error {
	normalizeEasyTierConfig(&c)
	if err := s.SetSetting(KeyETNetworkName, c.NetworkName); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETNetworkSecret, c.NetworkSecret); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETPeers, c.Peers); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETIPv4, c.IPv4); err != nil {
		return err
	}
	enabled := "0"
	if c.Enabled {
		enabled = "1"
	}
	if err := s.SetSetting(KeyETEnabled, enabled); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETImageTag, c.ImageTag); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETRPCPortal, c.RPCPortal); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETEnvFilePath, c.EnvFilePath); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETHostname, c.Hostname); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETExternalNode, c.ExternalNode); err != nil {
		return err
	}
	if err := s.SetSetting(KeyETProxyNetworks, c.ProxyNetworks); err != nil {
		return err
	}
	dhcp := "0"
	if c.DHCP {
		dhcp = "1"
	}
	if err := s.SetSetting(KeyETDHCP, dhcp); err != nil {
		return err
	}
	return nil
}

func normalizeEasyTierConfig(c *EasyTierConfig) {
	if c == nil {
		return
	}
	if c.DHCP {
		c.IPv4 = ""
	}
}

func normalizeProfiles(list []EasyTierProfile) []EasyTierProfile {
	out := make([]EasyTierProfile, 0, len(list))
	seen := make(map[string]bool, len(list))
	for _, p := range list {
		p.ID = strings.TrimSpace(p.ID)
		if p.ID == "" {
			continue
		}
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			p.Name = p.ID
		}
		normalizeEasyTierConfig(&p.Config)
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func hasProfileID(list []EasyTierProfile, id string) bool {
	_, ok := findProfile(list, id)
	return ok
}

func findProfile(list []EasyTierProfile, id string) (EasyTierProfile, bool) {
	for _, p := range list {
		if p.ID == id {
			return p, true
		}
	}
	return EasyTierProfile{}, false
}

func findProfileIndex(list []EasyTierProfile, id string) (int, bool) {
	for i, p := range list {
		if p.ID == id {
			return i, true
		}
	}
	return -1, false
}

func pickActiveProfileID(list []EasyTierProfile) string {
	if len(list) == 0 {
		return DefaultEasyTierProfileID
	}
	if _, ok := findProfile(list, DefaultEasyTierProfileID); ok {
		return DefaultEasyTierProfileID
	}
	return list[0].ID
}

func generateProfileID(name string, existing []EasyTierProfile) string {
	base := strings.ToLower(strings.TrimSpace(name))
	if base == "" {
		base = "profile"
	}
	var b strings.Builder
	for _, ch := range base {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
			continue
		}
		if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
			b.WriteByte('-')
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "profile"
	}
	candidate := id
	i := 2
	for hasProfileID(existing, candidate) {
		candidate = fmt.Sprintf("%s-%d", id, i)
		i++
	}
	return candidate
}
