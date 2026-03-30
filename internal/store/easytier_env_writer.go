package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

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
	// 容器内 EasyTier 需监听 RPC 地址；优先使用配置中的 RPCPortal，空则回退默认
	rpc := c.RPCPortal
	if rpc == "" {
		rpc = defaultEasyTierRPCPortal
	}
	b.WriteString("ET_RPC_PORTAL=")
	b.WriteString(escapeEnvValue(rpc))
	b.WriteString("\n")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, defaultDirectoryMode); err != nil {
		return fmt.Errorf("mkdir for easytier env: %w", err)
	}
	return os.WriteFile(path, b.Bytes(), privateFileMode)
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

