package frp

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/nginx"
)

type Status struct {
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
			st.ServerEndpoint = addr + ":" + strconv.Itoa(port)
		}
		st.HTTPGatewayEnabled = len(frpCfg.CustomDomains) > 0
		st.HTTPSGatewayEnabled = len(frpCfg.CustomDomains) > 0
	}
	if !enabled {
		return st
	}
	running, err := isProcessRunning(cfg.FrpPIDFile)
	if err != nil || !running {
		st.Message = "未连接"
		if st.LastError != "" {
			st.Message = st.LastError
		} else if hint := readLogHint(cfg.FrpLogPath()); hint != "" {
			st.Message = hint
		}
		return st
	}
	ok, logHint := logIndicatesConnected(cfg.FrpLogPath())
	if ok {
		st.Connected = true
		st.Message = "已连接"
		st.LastError = ""
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

func isProcessRunning(pidFile string) (bool, error) {
	pid, err := nginx.ReadPID(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return nginx.IsPIDAlive(pid), nil
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > max {
		lines = lines[len(lines)-max:]
	}
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
