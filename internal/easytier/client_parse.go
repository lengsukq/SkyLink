package easytier

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

func parseVersionFromCLI(s string) string {
	re := regexp.MustCompile(`(\d+\.\d+\.\d+(?:-\w+)?)`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
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

