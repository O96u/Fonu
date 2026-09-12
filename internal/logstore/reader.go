package logstore

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AccessEntry struct {
	Time          string  `json:"time"`
	Domain        string  `json:"domain"`
	Method        string  `json:"method"`
	Path          string  `json:"path"`
	Status        int     `json:"status"`
	ResponseTime  float64 `json:"response_time"`
	ClientIP      string  `json:"client_ip"`
	Upstream      string  `json:"upstream"`
	RequestLength int64   `json:"request_length,omitempty"`
	BytesSent     int64   `json:"bytes_sent,omitempty"`
}

type SystemEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Module  string `json:"module"`
	Message string `json:"message"`
}

var accessRe = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)$`)
var accessReWithBytes = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)\s+(\d+)\s+(\d+)$`)

func ParseAccess(line string) (AccessEntry, bool) {
	return parseAccess(line)
}

func ReadAccess(path string, limit int, keyword string, status int) ([]AccessEntry, error) {
	lines, err := tailLines(path, limit*4)
	if err != nil {
		return nil, err
	}
	var out []AccessEntry
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		entry, ok := parseAccess(lines[i])
		if !ok {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(entry.Domain+" "+entry.Path), strings.ToLower(keyword)) {
			continue
		}
		if status > 0 && entry.Status != status {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func AccessHostFilter(hosts []string) LineFilter {
	if len(hosts) == 0 {
		return nil
	}
	set := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host == "" {
			continue
		}
		set[host] = true
	}
	if len(set) == 0 {
		return nil
	}
	return func(line string) bool {
		entry, ok := parseAccess(line)
		if !ok {
			return false
		}
		return set[strings.ToLower(entry.Domain)]
	}
}

func parseAccess(line string) (AccessEntry, bool) {
	line = strings.TrimSpace(line)
	if m := accessReWithBytes.FindStringSubmatch(line); len(m) == 11 {
		status, _ := strconv.Atoi(m[5])
		rt, _ := strconv.ParseFloat(m[6], 64)
		reqLen, _ := strconv.ParseInt(m[9], 10, 64)
		bytesSent, _ := strconv.ParseInt(m[10], 10, 64)
		return AccessEntry{
			Time:          m[1],
			Domain:        m[2],
			Method:        m[3],
			Path:          m[4],
			Status:        status,
			ResponseTime:  rt,
			ClientIP:      m[7],
			Upstream:      m[8],
			RequestLength: reqLen,
			BytesSent:     bytesSent,
		}, true
	}
	m := accessRe.FindStringSubmatch(line)
	if len(m) != 9 {
		return AccessEntry{}, false
	}
	status, _ := strconv.Atoi(m[5])
	rt, _ := strconv.ParseFloat(m[6], 64)
	return AccessEntry{
		Time:         m[1],
		Domain:       m[2],
		Method:       m[3],
		Path:         m[4],
		Status:       status,
		ResponseTime: rt,
		ClientIP:     m[7],
		Upstream:     m[8],
	}, true
}

func ReadError(path string, limit int, keyword string) ([]string, error) {
	lines, err := tailLines(path, limit*2)
	if err != nil {
		return nil, err
	}
	var out []string
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

func isNoiseSystemEntry(entry SystemEntry) bool {
	return entry.Module == "HTTP" && strings.EqualFold(entry.Level, "INFO")
}

func ReadSystem(path string, limit int, level string, keyword string) ([]SystemEntry, error) {
	lines, err := tailLines(path, limit*20)
	if err != nil {
		return nil, err
	}
	var out []SystemEntry
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		entry, ok := parseSystem(lines[i])
		if !ok {
			continue
		}
		if isNoiseSystemEntry(entry) {
			continue
		}
		if level != "" && !strings.EqualFold(entry.Level, level) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(entry.Message+" "+entry.Module), strings.ToLower(keyword)) {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func parseSystem(line string) (SystemEntry, bool) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return SystemEntry{}, false
	}
	entry := SystemEntry{
		Time:    asString(raw["time"]),
		Level:   asString(raw["level"]),
		Message: asString(raw["msg"]),
		Module:  asString(raw["module"]),
	}
	if entry.Time == "" {
		entry.Time = time.Now().UTC().Format(time.RFC3339)
	}
	return entry, entry.Message != ""
}

func CountTodayAccess(path string) (total int, errors int, avgMs float64) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, 0
	}
	defer file.Close()

	today := time.Now().Format("2006-01-02")
	var sum float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, today) {
			continue
		}
		entry, ok := parseAccess(line)
		if !ok {
			continue
		}
		total++
		sum += entry.ResponseTime
		if entry.Status >= 400 {
			errors++
		}
	}
	if total > 0 {
		avgMs = sum / float64(total) * 1000
	}
	return total, errors, avgMs
}

func tailLines(path string, max int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > max {
			lines = lines[1:]
		}
	}
	return lines, scanner.Err()
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
