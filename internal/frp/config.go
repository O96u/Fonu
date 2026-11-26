package frp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

const (
	maskedToken = "********"
)

type Config struct {
	Enabled       bool       `json:"enabled"`
	ServerAddr    string     `json:"server_addr"`
	ServerPort    int        `json:"server_port"`
	AuthToken     string     `json:"auth_token"`
	HasAuthToken  bool       `json:"has_auth_token"`
	TLSEnabled    bool       `json:"tls_enabled"`
	CustomDomains []string   `json:"custom_domains"`
	TCPProxies    []TCPProxy `json:"tcp_proxies"`
}

type SaveInput struct {
	Enabled       bool
	ServerAddr    string
	ServerPort    int
	AuthToken     string
	TLSEnabled    bool
	CustomDomains []string
	TCPProxies    []TCPProxy
}

type Store struct {
	settings  *settings.Store
	secretBox *secret.Box
}

func NewStore(settingsStore *settings.Store, secretBox *secret.Box) *Store {
	return &Store{settings: settingsStore, secretBox: secretBox}
}

func (s *Store) Load(ctx context.Context) (Config, error) {
	enabled, _ := s.settings.GetBool(ctx, settings.KeyFRPEnabled)
	serverAddr, _ := s.settings.Get(ctx, settings.KeyFRPServerAddr)
	serverPort, _ := s.settings.GetInt(ctx, settings.KeyFRPServerPort)
	if serverPort <= 0 {
		serverPort = 7000
	}
	tlsEnabled, _ := s.settings.GetBool(ctx, settings.KeyFRPTLSEnabled)
	domains, err := s.loadDomains(ctx)
	if err != nil {
		return Config{}, err
	}
	tcpProxies, err := s.loadTCPProxies(ctx)
	if err != nil {
		return Config{}, err
	}
	encToken, _ := s.settings.Get(ctx, settings.KeyFRPAuthToken)
	hasToken := strings.TrimSpace(encToken) != ""
	token := ""
	if hasToken {
		token = maskedToken
	}
	return Config{
		Enabled:       enabled,
		ServerAddr:    serverAddr,
		ServerPort:    serverPort,
		AuthToken:     token,
		HasAuthToken:  hasToken,
		TLSEnabled:    tlsEnabled,
		CustomDomains: domains,
		TCPProxies:    tcpProxies,
	}, nil
}

func (s *Store) LoadRuntime(ctx context.Context) (Config, string, error) {
	cfg, err := s.Load(ctx)
	if err != nil {
		return Config{}, "", err
	}
	if !cfg.HasAuthToken {
		return cfg, "", nil
	}
	enc, _ := s.settings.Get(ctx, settings.KeyFRPAuthToken)
	plain, err := s.secretBox.Decrypt(enc)
	if err != nil {
		return cfg, "", secret.DecryptHint(err)
	}
	return cfg, plain, nil
}

func (s *Store) Save(ctx context.Context, in SaveInput, keepToken bool) error {
	if err := validateSaveInput(in, keepToken); err != nil {
		return err
	}
	port := in.ServerPort
	if port <= 0 {
		port = 7000
	}
	raw, err := json.Marshal(normalizeDomains(in.CustomDomains))
	if err != nil {
		return err
	}
	tcpProxies, err := prepareTCPProxies(in.TCPProxies)
	if err != nil {
		return err
	}
	tcpRaw, err := json.Marshal(tcpProxies)
	if err != nil {
		return err
	}
	values := map[string]string{
		settings.KeyFRPEnabled:       strconv.FormatBool(in.Enabled),
		settings.KeyFRPServerAddr:    normalizeServerAddr(in.ServerAddr),
		settings.KeyFRPServerPort:    strconv.Itoa(port),
		settings.KeyFRPTLSEnabled:    strconv.FormatBool(in.TLSEnabled),
		settings.KeyFRPCustomDomains: string(raw),
		settings.KeyFRPTCPProxies:    string(tcpRaw),
	}
	token := strings.TrimSpace(in.AuthToken)
	if token != "" && token != maskedToken {
		enc, err := s.secretBox.Encrypt(token)
		if err != nil {
			return err
		}
		values[settings.KeyFRPAuthToken] = enc
	} else if in.Enabled && !keepToken {
		return fmt.Errorf("请填写 FRP 认证 Token")
	}
	return s.settings.SetMany(ctx, values)
}

var snapshotKeys = []string{
	settings.KeyFRPEnabled, settings.KeyFRPServerAddr, settings.KeyFRPServerPort,
	settings.KeyFRPAuthToken, settings.KeyFRPTLSEnabled, settings.KeyFRPCustomDomains,
	settings.KeyFRPTCPProxies, settings.KeyFRPStartedAt,
}

func (s *Store) snapshot(ctx context.Context) (map[string]string, error) {
	values := make(map[string]string, len(snapshotKeys))
	for _, key := range snapshotKeys {
		value, err := s.settings.Get(ctx, key)
		if err != nil { return nil, err }
		values[key] = value
	}
	return values, nil
}

func (s *Store) restore(ctx context.Context, values map[string]string) error {
	return s.settings.SetMany(ctx, values)
}

func (s *Store) SetLastError(ctx context.Context, msg string) error {
	return s.settings.Set(ctx, settings.KeyFRPLastError, strings.TrimSpace(msg))
}

func (s *Store) LastError(ctx context.Context) string {
	v, _ := s.settings.Get(ctx, settings.KeyFRPLastError)
	return strings.TrimSpace(v)
}

func (s *Store) StartedAt(ctx context.Context) string {
	v, _ := s.settings.Get(ctx, settings.KeyFRPStartedAt)
	return strings.TrimSpace(v)
}

func (s *Store) SetStartedAt(ctx context.Context, value string) error {
	return s.settings.Set(ctx, settings.KeyFRPStartedAt, strings.TrimSpace(value))
}

func (s *Store) loadTCPProxies(ctx context.Context) ([]TCPProxy, error) {
	raw, _ := s.settings.Get(ctx, settings.KeyFRPTCPProxies)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var proxies []TCPProxy
	if err := json.Unmarshal([]byte(raw), &proxies); err != nil {
		return nil, fmt.Errorf("FRP TCP 隧道配置无效")
	}
	return normalizeTCPProxies(proxies), nil
}

func (s *Store) loadDomains(ctx context.Context) ([]string, error) {
	raw, _ := s.settings.Get(ctx, settings.KeyFRPCustomDomains)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var domains []string
	if err := json.Unmarshal([]byte(raw), &domains); err != nil {
		return nil, fmt.Errorf("FRP 域名配置无效")
	}
	return normalizeDomains(domains), nil
}

func normalizeServerAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	if idx := strings.IndexAny(addr, "/?#"); idx >= 0 {
		addr = addr[:idx]
	}
	return strings.Trim(strings.TrimSpace(addr), "[]")
}

func normalizeDomains(domains []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" || seen[strings.ToLower(d)] {
			continue
		}
		seen[strings.ToLower(d)] = true
		out = append(out, d)
	}
	return out
}

var domainLabel = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?package frp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

const (
	maskedToken = "********"
)

type Config struct {
	Enabled       bool       `json:"enabled"`
	ServerAddr    string     `json:"server_addr"`
	ServerPort    int        `json:"server_port"`
	AuthToken     string     `json:"auth_token"`
	HasAuthToken  bool       `json:"has_auth_token"`
	TLSEnabled    bool       `json:"tls_enabled"`
	CustomDomains []string   `json:"custom_domains"`
	TCPProxies    []TCPProxy `json:"tcp_proxies"`
}

type SaveInput struct {
	Enabled       bool
	ServerAddr    string
	ServerPort    int
	AuthToken     string
	TLSEnabled    bool
	CustomDomains []string
	TCPProxies    []TCPProxy
}

type Store struct {
	settings  *settings.Store
	secretBox *secret.Box
}

func NewStore(settingsStore *settings.Store, secretBox *secret.Box) *Store {
	return &Store{settings: settingsStore, secretBox: secretBox}
}

func (s *Store) Load(ctx context.Context) (Config, error) {
	enabled, _ := s.settings.GetBool(ctx, settings.KeyFRPEnabled)
	serverAddr, _ := s.settings.Get(ctx, settings.KeyFRPServerAddr)
	serverPort, _ := s.settings.GetInt(ctx, settings.KeyFRPServerPort)
	if serverPort <= 0 {
		serverPort = 7000
	}
	tlsEnabled, _ := s.settings.GetBool(ctx, settings.KeyFRPTLSEnabled)
	domains, err := s.loadDomains(ctx)
	if err != nil {
		return Config{}, err
	}
	tcpProxies, err := s.loadTCPProxies(ctx)
	if err != nil {
		return Config{}, err
	}
	encToken, _ := s.settings.Get(ctx, settings.KeyFRPAuthToken)
	hasToken := strings.TrimSpace(encToken) != ""
	token := ""
	if hasToken {
		token = maskedToken
	}
	return Config{
		Enabled:       enabled,
		ServerAddr:    serverAddr,
		ServerPort:    serverPort,
		AuthToken:     token,
		HasAuthToken:  hasToken,
		TLSEnabled:    tlsEnabled,
		CustomDomains: domains,
		TCPProxies:    tcpProxies,
	}, nil
}

func (s *Store) LoadRuntime(ctx context.Context) (Config, string, error) {
	cfg, err := s.Load(ctx)
	if err != nil {
		return Config{}, "", err
	}
	if !cfg.HasAuthToken {
		return cfg, "", nil
	}
	enc, _ := s.settings.Get(ctx, settings.KeyFRPAuthToken)
	plain, err := s.secretBox.Decrypt(enc)
	if err != nil {
		return cfg, "", secret.DecryptHint(err)
	}
	return cfg, plain, nil
}

func (s *Store) Save(ctx context.Context, in SaveInput, keepToken bool) error {
	if err := validateSaveInput(in, keepToken); err != nil {
		return err
	}
	port := in.ServerPort
	if port <= 0 {
		port = 7000
	}
	raw, err := json.Marshal(normalizeDomains(in.CustomDomains))
	if err != nil {
		return err
	}
	tcpProxies, err := prepareTCPProxies(in.TCPProxies)
	if err != nil {
		return err
	}
	tcpRaw, err := json.Marshal(tcpProxies)
	if err != nil {
		return err
	}
	values := map[string]string{
		settings.KeyFRPEnabled:       strconv.FormatBool(in.Enabled),
		settings.KeyFRPServerAddr:    normalizeServerAddr(in.ServerAddr),
		settings.KeyFRPServerPort:    strconv.Itoa(port),
		settings.KeyFRPTLSEnabled:    strconv.FormatBool(in.TLSEnabled),
		settings.KeyFRPCustomDomains: string(raw),
		settings.KeyFRPTCPProxies:    string(tcpRaw),
	}
	token := strings.TrimSpace(in.AuthToken)
	if token != "" && token != maskedToken {
		enc, err := s.secretBox.Encrypt(token)
		if err != nil {
			return err
		}
		values[settings.KeyFRPAuthToken] = enc
	} else if in.Enabled && !keepToken {
		return fmt.Errorf("请填写 FRP 认证 Token")
	}
	return s.settings.SetMany(ctx, values)
}

var snapshotKeys = []string{
	settings.KeyFRPEnabled, settings.KeyFRPServerAddr, settings.KeyFRPServerPort,
	settings.KeyFRPAuthToken, settings.KeyFRPTLSEnabled, settings.KeyFRPCustomDomains,
	settings.KeyFRPTCPProxies, settings.KeyFRPStartedAt,
}

func (s *Store) snapshot(ctx context.Context) (map[string]string, error) {
	values := make(map[string]string, len(snapshotKeys))
	for _, key := range snapshotKeys {
		value, err := s.settings.Get(ctx, key)
		if err != nil { return nil, err }
		values[key] = value
	}
	return values, nil
}

func (s *Store) restore(ctx context.Context, values map[string]string) error {
	return s.settings.SetMany(ctx, values)
}

func (s *Store) SetLastError(ctx context.Context, msg string) error {
	return s.settings.Set(ctx, settings.KeyFRPLastError, strings.TrimSpace(msg))
}

func (s *Store) LastError(ctx context.Context) string {
	v, _ := s.settings.Get(ctx, settings.KeyFRPLastError)
	return strings.TrimSpace(v)
}

func (s *Store) StartedAt(ctx context.Context) string {
	v, _ := s.settings.Get(ctx, settings.KeyFRPStartedAt)
	return strings.TrimSpace(v)
}

func (s *Store) SetStartedAt(ctx context.Context, value string) error {
	return s.settings.Set(ctx, settings.KeyFRPStartedAt, strings.TrimSpace(value))
}

func (s *Store) loadTCPProxies(ctx context.Context) ([]TCPProxy, error) {
	raw, _ := s.settings.Get(ctx, settings.KeyFRPTCPProxies)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var proxies []TCPProxy
	if err := json.Unmarshal([]byte(raw), &proxies); err != nil {
		return nil, fmt.Errorf("FRP TCP 隧道配置无效")
	}
	return normalizeTCPProxies(proxies), nil
}

func (s *Store) loadDomains(ctx context.Context) ([]string, error) {
	raw, _ := s.settings.Get(ctx, settings.KeyFRPCustomDomains)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var domains []string
	if err := json.Unmarshal([]byte(raw), &domains); err != nil {
		return nil, fmt.Errorf("FRP 域名配置无效")
	}
	return normalizeDomains(domains), nil
}

func normalizeServerAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	if idx := strings.IndexAny(addr, "/?#"); idx >= 0 {
		addr = addr[:idx]
	}
	return strings.Trim(strings.TrimSpace(addr), "[]")
}

func normalizeDomains(domains []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" || seen[strings.ToLower(d)] {
			continue
		}
		seen[strings.ToLower(d)] = true
		out = append(out, d)
	}
	return out
}

)

func validHost(value string) bool {
	if net.ParseIP(value) != nil { return true }
	if len(value) == 0 || len(value) > 253 { return false }
	for _, label := range strings.Split(strings.TrimSuffix(value, "."), ".") {
		if !domainLabel.MatchString(label) { return false }
	}
	return true
}

func validateSaveInput(in SaveInput, keepToken bool) error {
	addr := normalizeServerAddr(in.ServerAddr)
	if in.Enabled && addr == "" {
		return fmt.Errorf("请填写 FRP 服务器地址")
	}
	if addr != "" && !validHost(addr) { return fmt.Errorf("FRP 服务器地址无效，请单独填写端口") }
	for _, domain := range normalizeDomains(in.CustomDomains) {
		host := strings.TrimPrefix(domain, "*.")
		if !validHost(host) || net.ParseIP(host) != nil || !strings.Contains(host, ".") { return fmt.Errorf("无效穿透域名：%s", domain) }
	}
	if in.ServerPort < 0 || in.ServerPort > 65535 {
		return fmt.Errorf("FRP 服务器端口无效")
	}
	if len(normalizeTCPProxies(in.TCPProxies)) > 0 {
		if _, err := prepareTCPProxies(in.TCPProxies); err != nil {
			return err
		}
	}
	token := strings.TrimSpace(in.AuthToken)
	if token == "" || token == maskedToken {
		if in.Enabled && !keepToken {
			return fmt.Errorf("请填写 FRP 认证 Token")
		}
	}
	return nil
}
