package chinacidr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
)

const (
	defaultV4URL       = "https://raw.githubusercontent.com/17mon/china_ip_list/master/china_ip_list.txt"
	ghfastProxyPrefix  = "https://ghfast.top/"
)

// metowolf/iplist 已下线 CN/ipv6.txt；按优先级尝试可用源。
var defaultV6URLs = []string{
	"https://gaoyifan.github.io/china-operator-ip/china6.txt",
	"https://raw.githubusercontent.com/gaoyifan/china-operator-ip/china6.txt",
	"https://raw.githubusercontent.com/carrnot/china-ip-list/release/ipv6.txt",
}

type Service struct {
	cfg        config.Config
	settings   *settings.Store
	nginxMgr   *nginx.Manager
	proxyStore *proxy.Store
	logger     *slog.Logger
	updating   atomic.Bool
}

type Status struct {
	UpdatedAt    string `json:"updated_at,omitempty"`
	EntryCountV4 int    `json:"entry_count_v4"`
	EntryCountV6 int    `json:"entry_count_v6"`
	LastError    string `json:"last_error,omitempty"`
	Updating     bool   `json:"updating"`
	Ready        bool   `json:"ready"`
}

func New(cfg config.Config, settingsStore *settings.Store, nginxMgr *nginx.Manager, proxyStore *proxy.Store, logger *slog.Logger) *Service {
	return &Service{cfg: cfg, settings: settingsStore, nginxMgr: nginxMgr, proxyStore: proxyStore, logger: logger}
}

func (s *Service) Tick(ctx context.Context) {
	if err := s.Update(ctx); err != nil {
		s.logger.Warn("china cidr update failed", "module", "CHINA_CIDR", "error", err.Error())
		s.RecordLastError(ctx, err)
	}
}

func (s *Service) RecordLastError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	_ = s.settings.Set(ctx, settings.KeyChinaCIDRLastError, err.Error())
}

func (s *Service) Update(ctx context.Context) error {
	if !s.updating.CompareAndSwap(false, true) {
		return fmt.Errorf("正在更新中，请稍后再试")
	}
	defer s.updating.Store(false)

	v4URL, _ := s.settings.Get(ctx, settings.KeyChinaCIDRSourceV4)
	customV6URL, _ := s.settings.Get(ctx, settings.KeyChinaCIDRSourceV6)
	if isDeprecatedChinaCIDRSource(customV6URL) {
		customV6URL = ""
		_ = s.settings.Set(ctx, settings.KeyChinaCIDRSourceV6, "")
	}

	v4Lines, err := downloadFromSources(ctx, v4URL, defaultV4URL)
	if err != nil {
		return fmt.Errorf("下载 IPv4 CIDR 失败: %w", err)
	}
	v6Sources := append([]string{customV6URL}, defaultV6URLs...)
	v6Lines, err := downloadFromSources(ctx, v6Sources...)
	if err != nil {
		return fmt.Errorf("下载 IPv6 CIDR 失败: %w", err)
	}

	body, v4Count, v6Count, err := buildCIDRFile(v4Lines, v6Lines)
	if err != nil {
		return err
	}

	tmpPath := s.cfg.ChinaCIDRTempPath()
	if err := os.WriteFile(tmpPath, []byte(body), 0o644); err != nil {
		return err
	}

	rules, err := s.proxyStore.ListEnabled(ctx)
	if err != nil {
		return err
	}
	opts := s.loadGenerateOptions(ctx)
	validateOpts := opts
	validateOpts.ChinaCIDRPathOverride = tmpPath
	validateOpts.ChinaCIDRAvailable = true
	if err := s.nginxMgr.ValidateOnlyWithOptions(ctx, rules, nil, validateOpts); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("nginx 校验失败: %w", err)
	}

	finalPath := s.cfg.ChinaCIDRPath()
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return err
	}

	if _, err := s.nginxMgr.Apply(ctx, rules, nil); err != nil {
		return fmt.Errorf("nginx 重载失败: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_ = s.settings.Set(ctx, settings.KeyChinaCIDRUpdatedAt, now)
	_ = s.settings.Set(ctx, settings.KeyChinaCIDRCountV4, fmt.Sprintf("%d", v4Count))
	_ = s.settings.Set(ctx, settings.KeyChinaCIDRCountV6, fmt.Sprintf("%d", v6Count))
	_ = s.settings.Set(ctx, settings.KeyChinaCIDRLastError, "")

	s.logger.Info("china cidr updated", "module", "CHINA_CIDR", "v4", v4Count, "v6", v6Count)
	return nil
}

func (s *Service) Status(ctx context.Context) Status {
	updatedAt, _ := s.settings.Get(ctx, settings.KeyChinaCIDRUpdatedAt)
	v4, _ := s.settings.GetInt(ctx, settings.KeyChinaCIDRCountV4)
	v6, _ := s.settings.GetInt(ctx, settings.KeyChinaCIDRCountV6)
	lastErr, _ := s.settings.Get(ctx, settings.KeyChinaCIDRLastError)
	ready := updatedAt != "" && lastErr == "" && v4 > 0 && v6 > 0
	return Status{
		UpdatedAt:    updatedAt,
		EntryCountV4: v4,
		EntryCountV6: v6,
		LastError:    lastErr,
		Updating:     s.updating.Load(),
		Ready:        ready,
	}
}

func isDeprecatedChinaCIDRSource(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return false
	}
	return strings.Contains(url, "metowolf/iplist") || strings.Contains(url, "iplist/meta")
}

func ghfastProxyURL(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" || strings.HasPrefix(origin, ghfastProxyPrefix) {
		return ""
	}
	if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
		return ""
	}
	return ghfastProxyPrefix + origin
}

func expandDownloadSources(sources ...string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(url string) {
		url = strings.TrimSpace(url)
		if url == "" || isDeprecatedChinaCIDRSource(url) {
			return
		}
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}
	for _, raw := range sources {
		add(raw)
		if proxy := ghfastProxyURL(raw); proxy != "" {
			add(proxy)
		}
	}
	return out
}

func downloadFromSources(ctx context.Context, sources ...string) ([]string, error) {
	var tried []string
	var errs []string
	for _, url := range expandDownloadSources(sources...) {
		tried = append(tried, url)
		lines, err := downloadLines(ctx, url)
		if err == nil {
			return lines, nil
		}
		errs = append(errs, fmt.Sprintf("%s (%v)", shortSourceURL(url), err))
	}
	if len(tried) == 0 {
		return nil, fmt.Errorf("无可用的下载源")
	}
	return nil, fmt.Errorf("已尝试 %d 个源均失败: %s", len(tried), strings.Join(errs, "; "))
}

func shortSourceURL(url string) string {
	if len(url) <= 72 {
		return url
	}
	return url[:69] + "..."
}

func downloadLinesWithFallback(ctx context.Context, primary string, fallbacks ...string) ([]string, error) {
	return downloadFromSources(ctx, append([]string{primary}, fallbacks...)...)
}

func downloadLines(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func buildCIDRFile(v4Lines, v6Lines []string) (string, int, int, error) {
	var b strings.Builder
	v4Count := 0
	v6Count := 0
	for _, line := range v4Lines {
		cidr, err := normalizeCIDR(line)
		if err != nil {
			continue
		}
		if cidrTo4(cidr) == nil {
			continue
		}
		b.WriteString(cidr + " 1;\n")
		v4Count++
	}
	for _, line := range v6Lines {
		cidr, err := normalizeCIDR(line)
		if err != nil {
			continue
		}
		if cidrTo4(cidr) != nil {
			continue
		}
		b.WriteString(cidr + " 1;\n")
		v6Count++
	}
	if v4Count < 100 {
		return "", 0, 0, fmt.Errorf("IPv4 条目过少 (%d)，拒绝更新", v4Count)
	}
	if v6Count < 10 {
		return "", 0, 0, fmt.Errorf("IPv6 条目过少 (%d)，拒绝更新", v6Count)
	}
	return b.String(), v4Count, v6Count, nil
}

func normalizeCIDR(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "/") {
		if ip := net.ParseIP(raw); ip != nil {
			if v4 := ip.To4(); v4 != nil {
				return v4.String() + "/32", nil
			}
			return ip.String() + "/128", nil
		}
		return "", fmt.Errorf("invalid")
	}
	_, ipNet, err := net.ParseCIDR(raw)
	if err != nil {
		return "", err
	}
	return ipNet.String(), nil
}

func cidrTo4(cidr string) net.IP {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	return ip.To4()
}

func (s *Service) loadGenerateOptions(ctx context.Context) nginx.GenerateOptions {
	opts := nginx.GenerateOptions{}
	raw, _ := s.settings.Get(ctx, settings.KeyTrustedProxy)
	opts.TrustedProxy = nginx.ParseTrustedProxyJSON(raw)
	blRaw, _ := s.settings.Get(ctx, settings.KeyGlobalIPBlacklist)
	opts.GlobalIPBlacklist = ParseGlobalIPBlacklist(blRaw)
	return opts
}

func ParseGlobalIPBlacklist(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
	return list
}
