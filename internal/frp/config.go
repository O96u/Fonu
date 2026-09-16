package frp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

const (
	maskedToken = "********"
)

type Config struct {
	Enabled       bool     `json:"enabled"`
	ServerAddr    string   `json:"server_addr"`
	ServerPort    int      `json:"server_port"`
	AuthToken     string   `json:"auth_token"`
	HasAuthToken  bool     `json:"has_auth_token"`
	TLSEnabled    bool     `json:"tls_enabled"`
	CustomDomains []string `json:"custom_domains"`
}

type SaveInput struct {
	Enabled       bool
	ServerAddr    string
	ServerPort    int
	AuthToken     string
	TLSEnabled    bool
	CustomDomains []string
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
	if err := s.settings.SetBool(ctx, settings.KeyFRPEnabled, in.Enabled); err != nil {
		return err
	}
	if err := s.settings.Set(ctx, settings.KeyFRPServerAddr, normalizeServerAddr(in.ServerAddr)); err != nil {
		return err
	}
	port := in.ServerPort
	if port <= 0 {
		port = 7000
	}
	if err := s.settings.SetInt(ctx, settings.KeyFRPServerPort, port); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyFRPTLSEnabled, in.TLSEnabled); err != nil {
		return err
	}
	domains := normalizeDomains(in.CustomDomains)
	raw, err := json.Marshal(domains)
	if err != nil {
		return err
	}
	if err := s.settings.Set(ctx, settings.KeyFRPCustomDomains, string(raw)); err != nil {
		return err
	}
	token := strings.TrimSpace(in.AuthToken)
	if token != "" && token != maskedToken {
		enc, err := s.secretBox.Encrypt(token)
		if err != nil {
			return err
		}
		if err := s.settings.Set(ctx, settings.KeyFRPAuthToken, enc); err != nil {
			return err
		}
	} else if in.Enabled && !keepToken {
		return fmt.Errorf("请填写 FRP 认证 Token")
	}
	return nil
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
	return strings.TrimSpace(addr)
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

func validateSaveInput(in SaveInput, keepToken bool) error {
	if !in.Enabled {
		return nil
	}
	if strings.TrimSpace(in.ServerAddr) == "" {
		return fmt.Errorf("请填写 FRP 服务器地址")
	}
	if in.ServerPort < 0 || in.ServerPort > 65535 {
		return fmt.Errorf("FRP 服务器端口无效")
	}
	if len(normalizeDomains(in.CustomDomains)) == 0 {
		return fmt.Errorf("请至少填写一个穿透域名")
	}
	token := strings.TrimSpace(in.AuthToken)
	if token == "" || token == maskedToken {
		if !keepToken {
			return fmt.Errorf("请填写 FRP 认证 Token")
		}
	}
	return nil
}
