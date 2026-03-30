package easytier

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client 通过 easytier-cli 与 EasyTier 守护进程交互
type Client struct {
	cliPath   string
	rpcPortal string
	workDir   string // 非空时 runCLI 优先使用该目录（与 NewRPCClient / RuntimePaths 配合）
}

// NewClient 创建 EasyTier 客户端；rpcPortal 如 "easytier:15888"。
// 若已知 easytier-core 路径，优先使用 NewRPCClient(ResolveRuntimePaths(core), rpc)。
func NewClient(cliPath, rpcPortal string) *Client {
	if cliPath == "" {
		cliPath = CLICommand
	}
	return &Client{cliPath: cliPath, rpcPortal: rpcPortal}
}

// runCLI 在指定 RPC 地址下执行 easytier-cli 子命令，返回标准输出
func (c *Client) runCLI(args ...string) (stdout, stderr []byte, err error) {
	cmd := exec.Command(c.cliPath, args...)
	if c.rpcPortal != "" {
		cmd.Env = append(os.Environ(), "ET_RPC_PORTAL="+c.rpcPortal)
	} else {
		cmd.Env = os.Environ()
	}
	if wd := strings.TrimSpace(c.workDir); wd != "" {
		cmd.Dir = filepath.Clean(wd)
	} else if dir := filepath.Dir(c.cliPath); dir != "" && dir != "." {
		if fi, err := os.Stat(c.cliPath); err == nil && !fi.IsDir() {
			cmd.Dir = filepath.Clean(dir)
		}
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	stdout = outBuf.Bytes()
	stderr = errBuf.Bytes()
	if runErr != nil {
		return stdout, stderr, fmt.Errorf("%w: %s", runErr, bytes.TrimSpace(stderr))
	}
	return stdout, stderr, nil
}

// Version 返回当前连接的 EasyTier 版本（通过 CLI 或 --version）
func (c *Client) Version() (string, error) {
	// 尝试 easytier-cli --version 或 version 子命令
	out, _, err := c.runCLI("--version")
	if err != nil {
		out2, _, err2 := c.runCLI("version")
		if err2 != nil {
			return "", err
		}
		out = out2
	}
	s := strings.TrimSpace(string(out))
	// 可能形如 "easytier 2.2.3" 或 "2.2.3"
	if v := parseVersionFromCLI(s); v != "" {
		return v, nil
	}
	return s, nil
}

// CLISubcommandRaw 执行白名单内的 easytier-cli 子命令，返回原始 stdout/stderr（用于与官方文档对照排查）。
func (c *Client) CLISubcommandRaw(sub string) (stdout, stderr string, err error) {
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return "", "", fmt.Errorf("empty subcommand")
	}
	if sub == "version" {
		out, errOut, err := c.runCLI("--version")
		if err != nil {
			out2, errOut2, err2 := c.runCLI("version")
			if err2 != nil {
				return string(out), string(errOut), err
			}
			return string(out2), string(errOut2), nil
		}
		return string(out), string(errOut), nil
	}
	out, errOut, err := c.runCLI(sub)
	return string(out), string(errOut), err
}

// Status 获取 mesh 状态：本机 IP、peers、routes
func (c *Client) Status() (*Status, error) {
	st := &Status{OK: true, Peers: []Peer{}, Routes: []Route{}}
	// 本机版本
	if v, err := c.Version(); err == nil {
		st.Version = v
	}

	// peer 表
	peerOut, _, err := c.runCLI("peer")
	if err != nil {
		st.OK = false
		st.Error = err.Error()
		return st, nil
	}
	st.Peers = parsePeerTable(peerOut)
	// 从 peer 表推断本机：通常 cost=0 或首行为本机（以实际输出为准，此处取第一个 peer 的 version 与上面一致则可能为本机）
	// 若 easytier-cli node 存在则用其获取本机 IP
	nodeOut, _, _ := c.runCLI("node")
	if len(nodeOut) > 0 {
		st.SelfIPv4, st.SelfHost = parseNodeOutput(nodeOut)
	}
	if st.SelfIPv4 == "" && len(st.Peers) > 0 {
		// 启发式：本机多为 cost 最小或第一条
		st.SelfIPv4 = st.Peers[0].IPv4
		st.SelfHost = st.Peers[0].Hostname
	}

	// route 表
	routeOut, _, _ := c.runCLI("route")
	st.Routes = parseRouteTable(routeOut)
	return st, nil
}
