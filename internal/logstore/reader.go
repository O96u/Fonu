package logstore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AccessEntry struct {
	Time          string  `json:"time"`
	Domain        string  `json:"domain"`
	ServerPort    int     `json:"server_port,omitempty"`
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
var accessReWithPort = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\d+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)$`)
var accessReWithPortAndBytes = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\d+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)\s+(\d+)\s+(\d+)$`)

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
	endpoints := make([]AccessEndpoint, 0, len(hosts))
	for _, host := range hosts {
		endpoints = append(endpoints, AccessEndpoint{Host: host, Port: 0})
	}
	return AccessEndpointFilter(endpoints)
}

type AccessEndpoint struct {
	Host string
	Port int
}

func AccessEndpointFilter(endpoints []AccessEndpoint) LineFilter {
	if len(endpoints) == 0 {
		return nil
	}
	exact := make(map[string]bool, len(endpoints))
	hosts := make(map[string]bool, len(endpoints))
	for _, ep := range endpoints {
		host := normalizeAccessHost(ep.Host)
		if host == "" {
			continue
		}
		hosts[host] = true
		if ep.Port > 0 {
			exact[endpointMatchKey(host, ep.Port)] = true
		}
	}
	if len(hosts) == 0 {
		return nil
	}
	return func(line string) bool {
		entry, ok := parseAccess(line)
		if !ok {
			return false
		}
		host := normalizeAccessHost(entry.Domain)
		if entry.ServerPort > 0 {
			return exact[endpointMatchKey(host, entry.ServerPort)]
		}
		return hosts[host]
	}
}

func parseAccess(line string) (AccessEntry, bool) {
	line = strings.TrimSpace(line)
	if m := accessReWithPortAndBytes.FindStringSubmatch(line); len(m) == 12 {
		status, _ := strconv.Atoi(m[6])
		rt, _ := strconv.ParseFloat(m[7], 64)
		port, _ := strconv.Atoi(m[3])
		reqLen, _ := strconv.ParseInt(m[10], 10, 64)
		bytesSent, _ := strconv.ParseInt(m[11], 10, 64)
		return AccessEntry{
			Time:          m[1],
			Domain:        m[2],
			ServerPort:    port,
			Method:        m[4],
			Path:          m[5],
			Status:        status,
			ResponseTime:  rt,
			ClientIP:      m[8],
			Upstream:      m[9],
			RequestLength: reqLen,
			BytesSent:     bytesSent,
		}, true
	}
	if m := accessReWithPort.FindStringSubmatch(line); len(m) == 10 {
		status, _ := strconv.Atoi(m[6])
		rt, _ := strconv.ParseFloat(m[7], 64)
		port, _ := strconv.Atoi(m[3])
		return AccessEntry{
			Time:         m[1],
			Domain:       m[2],
			ServerPort:   port,
			Method:       m[4],
			Path:         m[5],
			Status:       status,
			ResponseTime: rt,
			ClientIP:     m[8],
			Upstream:     m[9],
		}, true
	}
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

func normalizeAccessHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.Index(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}

func endpointMatchKey(host string, port int) string {
	return fmt.Sprintf("%s:%d", normalizeAccessHost(host), port)
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

const hourlyBucketCount = 12

func isDashboardAccess(entry AccessEntry) bool {
	switch entry.Path {
	case "/favicon.ico", "/robots.txt":
		return false
	}
	if strings.HasPrefix(entry.Path, "/fonu-errors/") {
		return false
	}
	return true
}

func isSameCalendarDay(t, day time.Time) bool {
	lt := t.In(day.Location())
	dy, dm, dd := day.Date()
	ty, tm, td := lt.Date()
	return ty == dy && tm == dm && td == dd
}

// HourlyAccessCounts returns today's request counts on a fixed 0–24h axis (12 two-hour buckets).
// Future buckets (after the current time) stay zero.
func HourlyAccessCounts(path string) ([]int, []string) {
	return hourlyAccessCountsForDay(path, time.Now())
}

func hourlyAccessCountsForDay(path string, day time.Time) ([]int, []string) {
	now := time.Now()
	currentBucket := now.Hour() / 2
	if !isSameCalendarDay(day, now) {
		currentBucket = hourlyBucketCount - 1
	}
	counts := make([]int, hourlyBucketCount)
	labels := calendarHourlyLabels()

	file, err := os.Open(path)
	if err != nil {
		return counts, labels
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		entry, ok := parseAccess(scanner.Text())
		if !ok || !isDashboardAccess(entry) {
			continue
		}
		t, ok := parseAccessTime(entry.Time)
		if !ok || !isSameCalendarDay(t, day) {
			continue
		}
		bucket := t.In(day.Location()).Hour() / 2
		if bucket < 0 || bucket > currentBucket {
			continue
		}
		counts[bucket]++
	}
	return counts, labels
}

func calendarHourlyLabels() []string {
	labels := make([]string, hourlyBucketCount)
	for i := 0; i < hourlyBucketCount; i++ {
		labels[i] = fmt.Sprintf("%02d:00", i*2)
	}
	return labels
}

func parseAccessTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func isLocalToday(t time.Time, now time.Time) bool {
	return isSameCalendarDay(t, now)
}

func CountTodayAccess(path string) (total int, errors int, avgMs float64) {
	return CountAccessForDay(path, time.Now())
}

func CountAccessForDay(path string, day time.Time) (total int, errors int, avgMs float64) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, 0
	}
	defer file.Close()

	var sum float64
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		entry, ok := parseAccess(scanner.Text())
		if !ok || !isDashboardAccess(entry) {
			continue
		}
		t, ok := parseAccessTime(entry.Time)
		if !ok || !isSameCalendarDay(t, day) {
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
