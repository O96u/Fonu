package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
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

// catalog 仅用于关键词识别平台，不再作为“未检测到也展示”的固定清单。
var catalog = []catalogEntry{
	{Name: "飞牛 fnOS", Port: 5666, Platform: "fnos", Keywords: []string{"fnos", "飞牛", "fnos.cn", "飞牛影视"}},
	{Name: "AList", Port: 5244, Platform: "alist", Keywords: []string{"alist"}},
	{Name: "Jellyfin", Port: 8096, Platform: "jellyfin", Keywords: []string{"jellyfin"}},
	{Name: "PhotoPrism", Port: 2342, Platform: "photoprism", Keywords: []string{"photoprism"}},
	{Name: "qBittorrent", Port: 8080, Platform: "qbittorrent", Keywords: []string{"qbittorrent"}},
	{Name: "Portainer", Port: 9000, Platform: "portainer", Keywords: []string{"portainer"}},
	{Name: "Umami", Port: 3030, Platform: "umami", Keywords: []string{"umami"}},
	{Name: "MiniDLNA", Port: 8200, Platform: "minidlna", Keywords: []string{"minidlna"}},
}

var commonPorts = []int{
	80, 443, 3000, 3030, 3040, 5173, 5244, 5666, 6888, 6892, 6895,
	8008, 8013, 8014, 8015, 8080, 8096, 8200, 8888, 9000, 2342, 5000, 5600,
}

func New() *Service {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // 内网探测需接受自签证书
	return &Service{
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 2 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

func (s *Service) Scan(ctx context.Context, host string) []Result {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}

	type probeResult struct {
		result Result
		ok     bool
	}

	results := make([]probeResult, len(commonPorts))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16)

	for i, port := range commonPorts {
		wg.Add(1)
		go func(index, port int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, ok := s.probePort(ctx, host, port)
			results[index] = probeResult{result: result, ok: ok}
		}(i, port)
	}
	wg.Wait()

	out := make([]Result, 0, len(commonPorts))
	for _, item := range results {
		if item.ok {
			out = append(out, item.result)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Port < out[j].Port
	})
	return out
}

func (s *Service) probePort(ctx context.Context, host string, port int) (Result, bool) {
	if !s.portOpen(ctx, host, port) {
		return Result{}, false
	}

	upstream, title, body, server := s.probeHTTP(ctx, host, port)
	if upstream == "" {
		return Result{}, false
	}

	platform, suggestion := matchCatalog(title, body)
	name := displayName(title, server, port)
	if platform == "" {
		platform = slugPlatform(name, port)
	}

	return Result{
		Name:       name,
		Port:       port,
		Host:       host,
		Upstream:   upstream,
		Detected:   true,
		Platform:   platform,
		Title:      title,
		Suggestion: suggestion,
	}, true
}

func (s *Service) portOpen(ctx context.Context, host string, port int) bool {
	dialer := net.Dialer{Timeout: 800 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *Service) probeHTTP(ctx context.Context, host string, port int) (upstream, title, body, server string) {
	for _, scheme := range []string{"http", "https"} {
		url := fmt.Sprintf("%s://%s/", scheme, joinHostPort(host, port))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		resp.Body.Close()

		rawBody := string(bodyBytes)
		lowerBody := strings.ToLower(rawBody)
		title = extractTitle(lowerBody, rawBody)
		server = strings.TrimSpace(resp.Header.Get("Server"))
		return url, title, lowerBody, server
	}
	return "", "", "", ""
}

func joinHostPort(host string, port int) string {
	if strings.Contains(host, ":") {
		return fmt.Sprintf("[%s]:%d", host, port)
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func displayName(title, server string, port int) string {
	title = strings.TrimSpace(title)
	if title != "" {
		if len(title) > 80 {
			return title[:80]
		}
		return title
	}
	server = strings.TrimSpace(server)
	if server != "" {
		if len(server) > 80 {
			return server[:80]
		}
		return server
	}
	return fmt.Sprintf("Web 服务 :%d", port)
}

func matchCatalog(title, body string) (platform, suggestion string) {
	content := strings.ToLower(title + " " + body)
	for _, item := range catalog {
		if !keywordsMatch(item.Keywords, content) {
			continue
		}
		platform = item.Platform
		if item.Platform == "fnos" {
			suggestion = "检测到飞牛 fnOS 服务，可使用 nas.<域名> 创建反向代理"
		}
		return platform, suggestion
	}
	return "", ""
}

func keywordsMatch(keywords []string, content string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func slugPlatform(name string, port int) string {
	slug := strings.ToLower(name)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return fmt.Sprintf("web-%d", port)
	}
	return slug
}

func extractTitle(lowerBody, rawBody string) string {
	start := strings.Index(lowerBody, "<title>")
	end := strings.Index(lowerBody, "</title>")
	if start >= 0 && end > start {
		return strings.TrimSpace(rawBody[start+7 : end])
	}
	return ""
}
