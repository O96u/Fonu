package discovery

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	client *http.Client
}

type Result struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Host       string `json:"host"`
	Upstream   string `json:"upstream"`
	Detected   bool   `json:"detected"`
	Platform   string `json:"platform"`
	Title      string `json:"title,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type catalogEntry struct {
	Name     string
	Port     int
	Platform string
	Keywords []string
}

var catalog = []catalogEntry{
	{Name: "飞牛 fnOS", Port: 5666, Platform: "fnos", Keywords: []string{"fnos", "飞牛", "fnos.cn"}},
	{Name: "AList", Port: 5244, Platform: "alist", Keywords: []string{"alist"}},
	{Name: "Jellyfin", Port: 8096, Platform: "jellyfin", Keywords: []string{"jellyfin"}},
	{Name: "PhotoPrism", Port: 2342, Platform: "photoprism", Keywords: []string{"photoprism"}},
	{Name: "qBittorrent", Port: 8080, Platform: "qbittorrent", Keywords: []string{"qbittorrent"}},
	{Name: "Portainer", Port: 9000, Platform: "portainer", Keywords: []string{"portainer"}},
}

func New() *Service {
	return &Service{
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (s *Service) Scan(ctx context.Context, host string) []Result {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	out := make([]Result, 0, len(catalog))
	for _, item := range catalog {
		result := Result{
			Name:     item.Name,
			Port:     item.Port,
			Host:     host,
			Upstream: fmt.Sprintf("http://%s:%d", host, item.Port),
			Platform: item.Platform,
		}
		if !s.portOpen(ctx, host, item.Port) {
			out = append(out, result)
			continue
		}
		result.Detected = true
		title, body := s.probeHTTP(ctx, host, item.Port)
		result.Title = title
		if s.matchPlatform(item, title, body) {
			result.Platform = item.Platform
			if item.Platform == "fnos" {
				result.Suggestion = "检测到飞牛 fnOS 服务，可使用 nas.<域名> 创建反向代理"
			}
		}
		out = append(out, result)
	}
	return out
}

func (s *Service) portOpen(ctx context.Context, host string, port int) bool {
	dialer := net.Dialer{Timeout: 800 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *Service) probeHTTP(ctx context.Context, host string, port int) (string, string) {
	url := fmt.Sprintf("http://%s:%d/", host, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", ""
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	bodyBytes := make([]byte, 4096)
	n, _ := resp.Body.Read(bodyBytes)
	body := strings.ToLower(string(bodyBytes[:n]))
	title := extractTitle(body)
	return title, body
}

func (s *Service) matchPlatform(item catalogEntry, title, body string) bool {
	content := strings.ToLower(title + " " + body)
	for _, keyword := range item.Keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func extractTitle(body string) string {
	lower := strings.ToLower(body)
	start := strings.Index(lower, "<title>")
	end := strings.Index(lower, "</title>")
	if start >= 0 && end > start {
		return strings.TrimSpace(body[start+7 : end])
	}
	return ""
}
