package proxy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/validate"
)

type BasicAuthConfig struct {
	Enabled      bool   `json:"enabled,omitempty"`
	Username     string `json:"username,omitempty"`
	PasswordHash string `json:"password_hash,omitempty"` // bcrypt htpasswd line body
	HasPassword  bool   `json:"has_password,omitempty"`  // API response only
}

type RateLimitConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	Rate    int  `json:"rate,omitempty"`  // requests per second
	Burst   int  `json:"burst,omitempty"` // burst count
}

type ConnLimitConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	Max     int  `json:"max,omitempty"` // max concurrent connections per IP
}

type SecurityConfig struct {
	IPBlacklist       []string         `json:"ip_blacklist,omitempty"`
	IPWhitelist       []string         `json:"ip_whitelist,omitempty"`
	IPWhitelistMode   bool             `json:"ip_whitelist_mode,omitempty"`
	ChinaOnly         bool             `json:"china_only,omitempty"`
	BasicAuth         *BasicAuthConfig `json:"basic_auth,omitempty"`
	RateLimit         *RateLimitConfig `json:"rate_limit,omitempty"`
	ConnLimit         *ConnLimitConfig `json:"conn_limit,omitempty"`
	ProxySSLVerifyOff bool             `json:"proxy_ssl_verify_off,omitempty"`
	ProxyHostUpstream bool             `json:"proxy_host_upstream,omitempty"` // use $proxy_host instead of $host
	TLSMin13Only      bool             `json:"tls_min_13_only,omitempty"`
	SecurityHeaders   bool             `json:"security_headers,omitempty"`
}

type SecurityInput struct {
	IPBlacklist       []string
	IPWhitelist       []string
	IPWhitelistMode   *bool
	ChinaOnly         *bool
	BasicAuth         *BasicAuthInput
	RateLimit         *RateLimitConfig
	ConnLimit         *ConnLimitConfig
	ProxySSLVerifyOff *bool
	ProxyHostUpstream *bool
	TLSMin13Only      *bool
	SecurityHeaders   *bool
}

type BasicAuthInput struct {
	Enabled  *bool
	Username *string
	Password *string // plaintext on write only
}

// DefaultPrivateCIDRs are always exempt from china_only restrictions.
var DefaultPrivateCIDRs = []string{
	"10.0.0.0/8",
	"127.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"::1/128",
	"fc00::/7",
}

func ChinaBypassCIDRs(extra []string) []string {
	seen := make(map[string]bool, len(DefaultPrivateCIDRs)+len(extra))
	out := make([]string, 0, len(DefaultPrivateCIDRs)+len(extra))
	for _, cidr := range DefaultPrivateCIDRs {
		if !seen[cidr] {
			seen[cidr] = true
			out = append(out, cidr)
		}
	}
	for _, cidr := range extra {
		cidr = strings.TrimSpace(cidr)
		if cidr != "" && !seen[cidr] {
			seen[cidr] = true
			out = append(out, cidr)
		}
	}
	return out
}

func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{}
}

func ParseSecurityJSON(raw string) (SecurityConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return DefaultSecurityConfig(), nil
	}
	var cfg SecurityConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return SecurityConfig{}, fmt.Errorf("安全配置格式无效")
	}
	return cfg.Normalize(), nil
}

func (c SecurityConfig) Normalize() SecurityConfig {
	if c.RateLimit != nil && c.RateLimit.Enabled {
		if c.RateLimit.Rate <= 0 {
			c.RateLimit.Rate = 10
		}
		if c.RateLimit.Burst <= 0 {
			c.RateLimit.Burst = 20
		}
	}
	if c.ConnLimit != nil && c.ConnLimit.Enabled {
		if c.ConnLimit.Max <= 0 {
			c.ConnLimit.Max = 20
		}
	}
	return c
}

func (c SecurityConfig) ForAPI() SecurityConfig {
	out := c.Normalize()
	if out.BasicAuth != nil {
		hasPassword := out.BasicAuth.PasswordHash != ""
		out.BasicAuth = &BasicAuthConfig{
			Enabled:     out.BasicAuth.Enabled,
			Username:    out.BasicAuth.Username,
			HasPassword: hasPassword,
		}
	}
	return out
}

func (c SecurityConfig) MarshalJSON() ([]byte, error) {
	type alias SecurityConfig
	return json.Marshal(alias(c.Normalize()))
}

func (c SecurityConfig) ToJSON() (string, error) {
	b, err := c.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c SecurityConfig) BasicAuthEnabled() bool {
	return c.BasicAuth != nil && c.BasicAuth.Enabled && c.BasicAuth.Username != "" && c.BasicAuth.PasswordHash != ""
}

func (c SecurityConfig) RateLimitEnabled() bool {
	return c.RateLimit != nil && c.RateLimit.Enabled
}

func (c SecurityConfig) ConnLimitEnabled() bool {
	return c.ConnLimit != nil && c.ConnLimit.Enabled
}

func (c SecurityConfig) WhitelistEnabled() bool {
	return c.IPWhitelistMode && len(c.IPWhitelist) > 0
}

func MergeSecurity(current SecurityConfig, in SecurityInput) (SecurityConfig, error) {
	out := current.Normalize()

	if in.IPBlacklist != nil {
		out.IPBlacklist = in.IPBlacklist
	}
	if in.IPWhitelist != nil {
		out.IPWhitelist = in.IPWhitelist
	}
	if in.IPWhitelistMode != nil {
		out.IPWhitelistMode = *in.IPWhitelistMode
	}
	if in.ChinaOnly != nil {
		out.ChinaOnly = *in.ChinaOnly
	}
	if in.RateLimit != nil {
		out.RateLimit = in.RateLimit
	}
	if in.ConnLimit != nil {
		out.ConnLimit = in.ConnLimit
	}
	if in.ProxySSLVerifyOff != nil {
		out.ProxySSLVerifyOff = *in.ProxySSLVerifyOff
	}
	if in.ProxyHostUpstream != nil {
		out.ProxyHostUpstream = *in.ProxyHostUpstream
	}
	if in.TLSMin13Only != nil {
		out.TLSMin13Only = *in.TLSMin13Only
	}
	if in.SecurityHeaders != nil {
		out.SecurityHeaders = *in.SecurityHeaders
	}

	if in.BasicAuth != nil {
		if in.BasicAuth.Enabled != nil {
			if out.BasicAuth == nil {
				out.BasicAuth = &BasicAuthConfig{}
			}
			out.BasicAuth.Enabled = *in.BasicAuth.Enabled
		}
		if in.BasicAuth.Username != nil {
			if out.BasicAuth == nil {
				out.BasicAuth = &BasicAuthConfig{}
			}
			out.BasicAuth.Username = strings.TrimSpace(*in.BasicAuth.Username)
		}
		if in.BasicAuth.Password != nil && strings.TrimSpace(*in.BasicAuth.Password) != "" {
			if out.BasicAuth == nil {
				out.BasicAuth = &BasicAuthConfig{}
			}
			hash, err := validate.BasicAuthPasswordHash(out.BasicAuth.Username, *in.BasicAuth.Password)
			if err != nil {
				return SecurityConfig{}, err
			}
			out.BasicAuth.PasswordHash = hash
		}
	}

	return out, ValidateSecurity(out)
}

func ValidateSecurity(cfg SecurityConfig) error {
	cfg = cfg.Normalize()
	if err := validate.IPOrCIDRList(cfg.IPBlacklist); err != nil {
		return fmt.Errorf("IP 黑名单：%w", err)
	}
	if err := validate.IPOrCIDRList(cfg.IPWhitelist); err != nil {
		return fmt.Errorf("IP 白名单：%w", err)
	}
	if cfg.IPWhitelistMode && len(cfg.IPWhitelist) == 0 {
		return fmt.Errorf("启用白名单模式时至少添加一条 IP 或 CIDR")
	}
	if cfg.BasicAuth != nil && cfg.BasicAuth.Enabled {
		if err := validate.Username(cfg.BasicAuth.Username); err != nil {
			return fmt.Errorf("Basic Auth：%w", err)
		}
		if cfg.BasicAuth.PasswordHash == "" {
			return fmt.Errorf("Basic Auth 需要设置密码")
		}
	}
	if cfg.RateLimit != nil && cfg.RateLimit.Enabled {
		if cfg.RateLimit.Rate < 1 || cfg.RateLimit.Rate > 10000 {
			return fmt.Errorf("限流速率无效（1-10000 次/秒）")
		}
		if cfg.RateLimit.Burst < 1 || cfg.RateLimit.Burst > 100000 {
			return fmt.Errorf("限流突发值无效（1-100000）")
		}
	}
	if cfg.ConnLimit != nil && cfg.ConnLimit.Enabled {
		if cfg.ConnLimit.Max < 1 || cfg.ConnLimit.Max > 10000 {
			return fmt.Errorf("连接数限制无效（1-10000）")
		}
	}
	return nil
}

func ParseIPListText(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func FormatIPListText(list []string) string {
	return strings.Join(list, "\n")
}
