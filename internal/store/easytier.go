package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// EasyTier 配置在 settings 表中的 key 前缀
const (
	KeyETNetworkName    = "et.network_name"
	KeyETNetworkSecret  = "et.network_secret"
	KeyETPeers          = "et.peers"
	KeyETIPv4           = "et.ipv4"
	KeyETEnabled        = "et.enabled"
	KeyETImageTag       = "et.image_tag"
	KeyETRPCPortal      = "et.rpc_portal"
	KeyETEnvFilePath    = "et.env_file_path"
	KeyETHostname       = "et.hostname"
	KeyETExternalNode   = "et.external_node"
	KeyETProxyNetworks  = "et.proxy_networks"
	KeyETDHCP           = "et.dhcp"
	KeyETVPNPortal      = "et.vpn_portal"
)

// EasyTierConfig 供 API 和 env 文件使用的配置结构
type EasyTierConfig struct {
	NetworkName    string `json:"network_name"`
	NetworkSecret  string `json:"network_secret"`
	Peers          string `json:"peers"`
	IPv4           string `json:"ipv4"`
	Enabled        bool   `json:"enabled"`
	ImageTag       string `json:"image_tag"`
	RPCPortal      string `json:"rpc_portal"`
	EnvFilePath    string `json:"env_file_path"`
	Hostname       string `json:"hostname"`
	ExternalNode   string `json:"external_node"`
	ProxyNetworks  string `json:"proxy_networks"`
	DHCP           bool   `json:"dhcp"`
	VPNPortal      string `json:"vpn_portal"`
}

// GetEasyTierConfig 从 settings 表读取 EasyTier 配置
func (s *Store) GetEasyTierConfig() (*EasyTierConfig, error) {
	c := &EasyTierConfig{}
	var err error
	if c.NetworkName, err = s.GetSetting(KeyETNetworkName); err != nil {
		return nil, err
	}
	if c.NetworkSecret, err = s.GetSetting(KeyETNetworkSecret); err != nil {
		return nil, err
	}
	if c.Peers, err = s.GetSetting(KeyETPeers); err != nil {
		return nil, err
	}
	if c.IPv4, err = s.GetSetting(KeyETIPv4); err != nil {
		return nil, err
	}
	if v, err := s.GetSetting(KeyETEnabled); err != nil {
		return nil, err
	} else {
		c.Enabled = v == "1" || v == "true"
	}
	if c.ImageTag, err = s.GetSetting(KeyETImageTag); err != nil {
		return nil, err
	}
	if c.RPCPortal, err = s.GetSetting(KeyETRPCPortal); err != nil {
		return nil, err
	}
	if c.EnvFilePath, err = s.GetSetting(KeyETEnvFilePath); err != nil {
		return nil, err
	}
	if c.Hostname, err = s.GetSetting(KeyETHostname); err != nil {
		return nil, err
	}
	if c.ExternalNode, err = s.GetSetting(KeyETExternalNode); err != nil {
		return nil, err
	}
	if c.ProxyNetworks, err = s.GetSetting(KeyETProxyNetworks); err != nil {
		return nil, err
	}
	if v, err := s.GetSetting(KeyETDHCP); err != nil {
		return nil, err
	} else {
		c.DHCP = v == "1" || v == "true"
	}
	if c.VPNPortal, err = s.GetSetting(KeyETVPNPortal); err != nil {
		return nil, err
	}
	return c, nil
}

// SetEasyTierConfig 将 EasyTier 配置写入 settings 表
func (s *Store) SetEasyTierConfig(c *EasyTierConfig) error {
	if c == nil {
		return nil
	}
	// 若启用 DHCP，则不再保留静态 IPv4，避免与 ET_DHCP 同时生效产生歧义
	if c.DHCP {
		c.IPv4 = ""
	}
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
	if err := s.SetSetting(KeyETVPNPortal, c.VPNPortal); err != nil {
		return err
	}
	return nil
}

// WriteEasyTierEnv 将当前 EasyTier 配置写入 env 文件。
// 该功能用于在需要时将配置共享给外部进程（例如独立运行的 EasyTier 或容器），
// 对于由 SkyLink 直接拉起并管理的守护进程模式并非必需。
func (s *Store) WriteEasyTierEnv(path string, c *EasyTierConfig) error {
	if path == "" || c == nil {
		return nil
	}
	var b bytes.Buffer
	if c.ImageTag != "" {
		b.WriteString("EASYTIER_IMAGE_TAG=")
		b.WriteString(escapeEnvValue(c.ImageTag))
		b.WriteString("\n")
	}
	if c.NetworkName != "" {
		b.WriteString("ET_NETWORK_NAME=")
		b.WriteString(escapeEnvValue(c.NetworkName))
		b.WriteString("\n")
	}
	if c.NetworkSecret != "" {
		b.WriteString("ET_NETWORK_SECRET=")
		b.WriteString(escapeEnvValue(c.NetworkSecret))
		b.WriteString("\n")
	}
	if c.Peers != "" {
		b.WriteString("ET_PEERS=")
		b.WriteString(escapeEnvValue(c.Peers))
		b.WriteString("\n")
	}
	if c.IPv4 != "" {
		b.WriteString("ET_IPV4=")
		b.WriteString(escapeEnvValue(c.IPv4))
		b.WriteString("\n")
	}
	if c.Hostname != "" {
		b.WriteString("ET_HOSTNAME=")
		b.WriteString(escapeEnvValue(c.Hostname))
		b.WriteString("\n")
	}
	if c.ExternalNode != "" {
		b.WriteString("ET_EXTERNAL_NODE=")
		b.WriteString(escapeEnvValue(c.ExternalNode))
		b.WriteString("\n")
	}
	if c.ProxyNetworks != "" {
		b.WriteString("ET_PROXY_NETWORKS=")
		b.WriteString(escapeEnvValue(c.ProxyNetworks))
		b.WriteString("\n")
	}
	// easytier-core --dhcp 仅接受 true/false，不能为 1/0
	b.WriteString("ET_DHCP=")
	if c.DHCP {
		b.WriteString("true\n")
	} else {
		b.WriteString("false\n")
	}
	if c.VPNPortal != "" {
		b.WriteString("ET_VPN_PORTAL=")
		b.WriteString(escapeEnvValue(c.VPNPortal))
		b.WriteString("\n")
	}
	// 容器内 EasyTier 需监听 RPC 地址；优先使用配置中的 RPCPortal，空则回退默认
	rpc := c.RPCPortal
	if rpc == "" {
		rpc = "0.0.0.0:15888"
	}
	b.WriteString("ET_RPC_PORTAL=")
	b.WriteString(escapeEnvValue(rpc))
	b.WriteString("\n")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir for easytier env: %w", err)
	}
	return os.WriteFile(path, b.Bytes(), 0600)
}

func escapeEnvValue(v string) string {
	if v == "" {
		return ""
	}
	// 若包含空格或特殊字符则加引号
	for _, c := range v {
		if c == ' ' || c == '"' || c == '\'' || c == '\n' || c == '\\' {
			return strconv.Quote(v)
		}
	}
	return v
}
