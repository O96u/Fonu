package frp

import (
	"fmt"
	"math/rand"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	tcpProxyNameMaxLen = 32
	tcpRemotePortMin   = 1024
)

var tcpProxyNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type TCPProxy struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	LocalIP    string `json:"local_ip"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Enabled    bool   `json:"enabled"`
}

func prepareTCPProxies(proxies []TCPProxy) ([]TCPProxy, error) {
	proxies = assignTCPRemotePorts(normalizeTCPProxies(proxies))
	if err := validateTCPProxies(proxies); err != nil {
		return nil, err
	}
	return proxies, nil
}

func assignTCPRemotePorts(proxies []TCPProxy) []TCPProxy {
	used := make(map[int]bool, len(proxies))
	for _, p := range proxies {
		if p.RemotePort > 0 {
			used[p.RemotePort] = true
		}
	}
	out := make([]TCPProxy, len(proxies))
	for i, p := range proxies {
		out[i] = p
		if p.RemotePort != 0 {
			continue
		}
		out[i].RemotePort = nextTCPRemotePort(used)
		used[out[i].RemotePort] = true
	}
	return out
}

func nextTCPRemotePort(used map[int]bool) int {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for attempt := 0; attempt < 256; attempt++ {
		port := tcpRemotePortMin + rng.Intn(65535-tcpRemotePortMin+1)
		if !used[port] {
			return port
		}
	}
	for port := tcpRemotePortMin; port <= 65535; port++ {
		if !used[port] {
			return port
		}
	}
	return tcpRemotePortMin
}

func normalizeTCPProxies(proxies []TCPProxy) []TCPProxy {
	var out []TCPProxy
	seenID := make(map[string]bool)
	for _, p := range proxies {
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			continue
		}
		p.LocalIP = strings.TrimSpace(p.LocalIP)
		if p.LocalIP == "" {
			p.LocalIP = "127.0.0.1"
		}
		p.ID = strings.TrimSpace(p.ID)
		if p.ID == "" || seenID[p.ID] {
			p.ID = uuid.NewString()
		}
		seenID[p.ID] = true
		out = append(out, p)
	}
	return out
}

func validateTCPProxies(proxies []TCPProxy) error {
	proxies = normalizeTCPProxies(proxies)
	remotePorts := make(map[int]string)
	names := make(map[string]string)
	for i, p := range proxies {
		label := fmt.Sprintf("TCP 隧道「%s」", p.Name)
		if len(p.Name) > tcpProxyNameMaxLen {
			return fmt.Errorf("%s：名称过长（最多 %d 个字符）", label, tcpProxyNameMaxLen)
		}
		if !tcpProxyNamePattern.MatchString(p.Name) {
			return fmt.Errorf("%s：名称仅允许字母、数字、下划线和连字符", label)
		}
		if other, ok := names[strings.ToLower(p.Name)]; ok {
			return fmt.Errorf("TCP 隧道名称重复：%s 与 %s", p.Name, other)
		}
		names[strings.ToLower(p.Name)] = p.Name
		if ip := net.ParseIP(p.LocalIP); ip == nil {
			return fmt.Errorf("%s：本地地址无效", label)
		}
		if p.LocalPort < 1 || p.LocalPort > 65535 {
			return fmt.Errorf("%s：本地端口无效", label)
		}
		if p.RemotePort != 0 && (p.RemotePort < tcpRemotePortMin || p.RemotePort > 65535) {
			return fmt.Errorf("%s：远程端口需为 0（自动分配）或 %d–65535", label, tcpRemotePortMin)
		}
		if p.RemotePort == 0 {
			continue
		}
		if other, ok := remotePorts[p.RemotePort]; ok {
			return fmt.Errorf("TCP 远程端口 %d 重复（%s 与 %s）", p.RemotePort, p.Name, other)
		}
		remotePorts[p.RemotePort] = p.Name
		proxies[i] = p
	}
	return nil
}

func enabledTCPProxies(proxies []TCPProxy) []TCPProxy {
	var out []TCPProxy
	for _, p := range normalizeTCPProxies(proxies) {
		if p.Enabled {
			out = append(out, p)
		}
	}
	return out
}

func tcpRemotePorts(proxies []TCPProxy) []int {
	ports := make([]int, 0, len(proxies))
	for _, p := range enabledTCPProxies(proxies) {
		if p.RemotePort <= 0 {
			continue
		}
		ports = append(ports, p.RemotePort)
	}
	sort.Ints(ports)
	return ports
}

func sanitizeTCPProxyName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	out := b.String()
	if out == "" {
		out = "tunnel"
	}
	if len(out) > tcpProxyNameMaxLen {
		out = out[:tcpProxyNameMaxLen]
	}
	return out
}

func frpcTCPProxyName(name string) string {
	return "fonu-tcp-" + sanitizeTCPProxyName(name)
}

func hasActiveRoutes(cfg Config) bool {
	if len(normalizeDomains(cfg.CustomDomains)) > 0 {
		return true
	}
	return len(enabledTCPProxies(cfg.TCPProxies)) > 0
}
