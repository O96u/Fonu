package frp

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/config"
)

type Status struct {
	ProcessRunning bool `json:"process_running"`
	TunnelStates map[string]string `json:"tunnel_states"`
	HTTPGatewayState string `json:"http_gateway_state"`
	HTTPSGatewayState string `json:"https_gateway_state"`
	Enabled             bool   `json:"enabled"`
	Connected           bool   `json:"connected"`
	Message             string `json:"message"`
	LastError           string `json:"last_error,omitempty"`
	ServerEndpoint      string `json:"server_endpoint,omitempty"`
	ConnectedAt         string `json:"connected_at,omitempty"`
	UptimeSeconds       int64  `json:"uptime_seconds"`
	ClientVersion       string `json:"client_version,omitempty"`
	HTTPGatewayEnabled  bool   `json:"http_gateway_enabled"`
	HTTPSGatewayEnabled bool   `json:"https_gateway_enabled"`
	SyncedDomainCount   int    `json:"synced_domain_count"`
}

func BuildStatus(cfg config.Config, frpCfg Config, enabled bool, lastError, startedAt, clientVersion string) Status {
	st := Status{
		Enabled:           enabled,
		TunnelStates: make(map[string]string),
		HTTPGatewayState: "unknown",
		HTTPSGatewayState: "unknown",
		Connected:         false,
		Message:           "未启用",
		LastError:         strings.TrimSpace(lastError),
		ClientVersion:     clientVersion,
		SyncedDomainCount: len(frpCfg.CustomDomains),
	}
	if enabled {
		addr := strings.TrimSpace(frpCfg.ServerAddr)
		if addr != "" {
			port := frpCfg.ServerPort
			if port <= 0 {
				port = 7000
			}
			st.ServerEndpoint = net.JoinHostPort(strings.Trim(addr, "[]"), strconv.Itoa(port))
		}
		st.HTTPGatewayEnabled = len(frpCfg.CustomDomains) > 0
		st.HTTPSGatewayEnabled = len(frpCfg.CustomDomains) > 0
	}
	if !enabled {
		return st
	}
	running, err := isProcessRunning(cfg)
	if err != nil || !running {
		st.Message = "未连接"
		if st.LastError != "" {
			st.Message = st.LastError
		} else if hint := readLogHint(cfg.FrpLogPath()); hint != "" {
			st.Message = hint
		}
		return st
	}
	st.ProcessRunning = true
	applyConnectedTiming(&st, startedAt)
	ok, logHint, registrations := sessionStatus(tailLines(cfg.FrpLogPath(), 1000))
	if ok {
		st.Connected = true
		st.Message = "已连接"
		st.HTTPGatewayState = registrationState(registrations, "fonu-nginx-http")
		st.HTTPSGatewayState = registrationState(registrations, "fonu-nginx-https")
		for _, tunnel := range frpCfg.TCPProxies {
			if tunnel.Enabled { st.TunnelStates[tunnel.ID] = registrationState(registrations, frpcTunnelProxyName(tunnel.Protocol, tunnel.Name)) }
		}
		applyConnectedTiming(&st, startedAt)
		return st
	}
	st.Message = "连接中"
	if logHint != "" {
		st.Message = logHint
	}
	if st.LastError != "" {
		st.Message = st.LastError
	}
	return st
}

func applyConnectedTiming(st *Status, startedAt string) {
	st.ConnectedAt = strings.TrimSpace(startedAt)
	if st.ConnectedAt == "" {
		return
	}
	t, err := time.Parse(time.RFC3339, st.ConnectedAt)
	if err != nil {
		return
	}
	st.UptimeSeconds = int64(time.Since(t).Seconds())
	if st.UptimeSeconds < 0 {
		st.UptimeSeconds = 0
	}
}

func isProcessRunning(cfg config.Config) (bool, error) {
	pid, err := readPIDFile(cfg.FrpPIDFile)
	if err != nil {
		if os.IsNotExist(err) { return false, nil }
		return false, err
	}
	return isOwnedProcess(cfg, pid), nil
}

func registrationState(states map[string]string, name string) string {
	if state, ok := states[strings.ToLower(name)]; ok { return state }
	return "unknown"
}

// Log evidence is scoped to this launch, not historical sessions. A missing
// marker (e.g. log rotation) means unknown rather than a fabricated success.
func sessionStatus(lines []string) (bool, string, map[string]string) {
	states := make(map[string]string)
	connected, current := false, false
	hint := ""
	for _, raw := range lines {
		if strings.TrimSpace(raw) == sessionMarker {
			current, connected, hint = true, false, ""
			clear(states)
			continue
		}
		if !current { continue }
		line := strings.ToLower(raw)
		switch {
		case strings.Contains(line, "login to server success"):
			connected, hint = true, ""
			clear(states)
		case strings.Contains(line, "connect to server error"), strings.Contains(line, "authorization failed"), strings.Contains(line, "token in login doesn't match"), strings.Contains(line, "control writer is closing"), strings.Contains(line, "control reader is closing"), strings.Contains(line, "session shutdown"), strings.Contains(line, "control connection closed"), strings.Contains(line, "heartbeat timeout"):
			connected, hint = false, sanitizeLogLine(raw)
			clear(states)
		}
		if !connected { continue }
		start := strings.Index(line, "[fonu-")
		if start < 0 { continue }
		end := strings.IndexByte(line[start:], ']')
		if end < 0 { continue }
		name := line[start+1:start+end]
		switch {
		case strings.Contains(line, "start proxy success"):
			states[name] = "registered"
		case strings.Contains(line, "start error"), strings.Contains(line, "start proxy error"), strings.Contains(line, "proxy closed"):
			states[name] = "failed"
		}
	}
	return connected, hint, states
}

func readLogHint(path string) string {
	lines := tailLines(path, 40)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.ToLower(lines[i])
		if strings.Contains(line, "login to server success") {
			return ""
		}
		if strings.Contains(line, "connect to server error") ||
			strings.Contains(line, "authorization failed") ||
			strings.Contains(line, "token in login doesn't match") {
			return sanitizeLogLine(lines[i])
		}
	}
	return ""
}

func logIndicatesConnected(path string) (bool, string) {
	lines := tailLines(path, 80)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.ToLower(lines[i])
		if strings.Contains(line, "login to server success") {
			return true, ""
		}
		if strings.Contains(line, "connect to server error") ||
			strings.Contains(line, "authorization failed") {
			return false, sanitizeLogLine(lines[i])
		}
	}
	return false, ""
}

func tailLines(path string, max int) []string {
	lines, _ := ReadLogTail(path, max)
	return lines
}

func sanitizeLogLine(line string) string {
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, "]"); idx >= 0 && idx+1 < len(line) {
		line = strings.TrimSpace(line[idx+1:])
	}
	if len(line) > 200 {
		line = line[:200] + "…"
	}
	return line
}
