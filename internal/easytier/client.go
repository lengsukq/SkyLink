package easytier

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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

func parseVersionFromCLI(s string) string {
	// 匹配语义化版本
	re := regexp.MustCompile(`(\d+\.\d+\.\d+(?:-\w+)?)`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
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

func parsePeerTable(out []byte) []Peer {
	var peers []Peer
	sc := bufio.NewScanner(bytes.NewReader(out))
	var header []string
	ipv4Col := -1
	hostCol := -1
	versionCol := -1
	costCol := -1
	latencyCol := -1
	tunnelCol := -1
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		cols := splitTableRow(line)
		if len(cols) == 0 {
			continue
		}
		if header == nil {
			header = cols
			for i, h := range header {
				lower := strings.ToLower(strings.TrimSpace(h))
				if strings.HasPrefix(lower, "ipv4") {
					ipv4Col = i
				}
				if lower == "hostname" {
					hostCol = i
				}
				if lower == "version" {
					versionCol = i
				}
				if lower == "cost" {
					costCol = i
				}
				if lower == "latency" || lower == "latency_ms" || lower == "lat" {
					latencyCol = i
				}
				if lower == "tunnel" || lower == "protocol" {
					tunnelCol = i
				}
			}
			continue
		}
		p := Peer{}
		if ipv4Col >= 0 && ipv4Col < len(cols) {
			p.IPv4 = strings.TrimSpace(cols[ipv4Col])
		}
		if hostCol >= 0 && hostCol < len(cols) {
			p.Hostname = strings.TrimSpace(cols[hostCol])
		}
		if versionCol >= 0 && versionCol < len(cols) {
			p.Version = strings.TrimSpace(cols[versionCol])
		}
		if costCol >= 0 && costCol < len(cols) {
			if v, err := strconv.Atoi(strings.TrimSpace(cols[costCol])); err == nil {
				p.Cost = v
			}
		}
		if latencyCol >= 0 && latencyCol < len(cols) {
			if v, err := strconv.ParseFloat(strings.TrimSpace(cols[latencyCol]), 64); err == nil {
				p.Latency = v
			}
		}
		if tunnelCol >= 0 && tunnelCol < len(cols) {
			p.Tunnel = strings.TrimSpace(cols[tunnelCol])
		}
		if p.IPv4 != "" && isIPv4(p.IPv4) {
			peers = append(peers, p)
		}
	}
	return peers
}

func parseRouteTable(out []byte) []Route {
	var routes []Route
	sc := bufio.NewScanner(bytes.NewReader(out))
	var header []string
	ipv4Col := -1
	hostCol := -1
	proxyCIDRsCol := -1
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		cols := splitTableRow(line)
		if header == nil {
			header = cols
			for i, h := range header {
				name := strings.ToLower(strings.TrimSpace(h))
				switch name {
				case "ipv4":
					ipv4Col = i
				case "hostname":
					hostCol = i
				case "proxy_cidrs":
					proxyCIDRsCol = i
				}
			}
			continue
		}
		r := Route{}
		if ipv4Col >= 0 && ipv4Col < len(cols) {
			r.IPv4 = strings.TrimSpace(cols[ipv4Col])
		}
		if hostCol >= 0 && hostCol < len(cols) {
			r.Hostname = strings.TrimSpace(cols[hostCol])
		}
		if proxyCIDRsCol >= 0 && proxyCIDRsCol < len(cols) {
			r.ProxyCIDRs = strings.TrimSpace(cols[proxyCIDRsCol])
		}
		if r.IPv4 != "" {
			routes = append(routes, r)
		}
	}
	return routes
}

func parseNodeOutput(out []byte) (ipv4, hostname string) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if idx := strings.Index(line, "ipv4"); idx >= 0 {
			rest := line[idx+4:]
			if v := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`).FindString(rest); v != "" {
				ipv4 = v
			}
		}
		if idx := strings.Index(line, "hostname"); idx >= 0 {
			rest := strings.TrimSpace(line[idx+8:])
			hostname = strings.Split(rest, " ")[0]
		}
	}
	return ipv4, hostname
}

func splitTableRow(line string) []string {
	// 先按制表符分，否则按连续空白分
	if strings.Contains(line, "\t") {
		return strings.Split(line, "\t")
	}
	return regexp.MustCompile(`\s{2,}`).Split(strings.TrimSpace(line), -1)
}

func isIPv4(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 3 {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// Release 表示 GitHub 上一个 release 的简要信息
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// FetchLatestRelease 请求 GitHub API 获取最新 release tag
func FetchLatestRelease() (tagName, htmlURL string, err error) {
	req, err := http.NewRequest(http.MethodGet, GitHubReleasesAPI, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github api: %s", resp.Status)
	}
	var v struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", "", err
	}
	return v.TagName, v.HTMLURL, nil
}

// FetchReleases 请求 GitHub API 获取 releases 列表，用于版本下拉；perPage 建议 30
func FetchReleases(perPage int) ([]Release, error) {
	if perPage <= 0 {
		perPage = 30
	}
	url := fmt.Sprintf("%s?per_page=%d", GitHubReleasesListAPI, perPage)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases api: %s", resp.Status)
	}
	var list []Release
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}
