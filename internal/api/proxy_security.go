package api

import (
	"github.com/fonu/fonu/internal/proxy"
)

type securityRequest struct {
	IPBlacklist       []string               `json:"ip_blacklist,omitempty"`
	IPWhitelist       []string               `json:"ip_whitelist,omitempty"`
	IPWhitelistMode   *bool                  `json:"ip_whitelist_mode,omitempty"`
	IPBlacklistText   *string                `json:"ip_blacklist_text,omitempty"`
	IPWhitelistText   *string                `json:"ip_whitelist_text,omitempty"`
	ChinaOnly         *bool                  `json:"china_only,omitempty"`
	BasicAuth         *basicAuthRequest      `json:"basic_auth,omitempty"`
	RateLimit         *proxy.RateLimitConfig `json:"rate_limit,omitempty"`
	ConnLimit         *proxy.ConnLimitConfig `json:"conn_limit,omitempty"`
	ProxySSLVerifyOff *bool                  `json:"proxy_ssl_verify_off,omitempty"`
	ProxyHostUpstream *bool                  `json:"proxy_host_upstream,omitempty"`
	TLSMin13Only      *bool                  `json:"tls_min_13_only,omitempty"`
	SecurityHeaders   *bool                  `json:"security_headers,omitempty"`
}

type basicAuthRequest struct {
	Enabled  *bool   `json:"enabled,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

func (req *proxyRequest) securityInput() proxy.SecurityInput {
	if req.Security == nil {
		return proxy.SecurityInput{}
	}
	s := req.Security
	in := proxy.SecurityInput{
		IPBlacklist:       s.IPBlacklist,
		IPWhitelist:       s.IPWhitelist,
		IPWhitelistMode:   s.IPWhitelistMode,
		ChinaOnly:         s.ChinaOnly,
		RateLimit:         s.RateLimit,
		ConnLimit:         s.ConnLimit,
		ProxySSLVerifyOff: s.ProxySSLVerifyOff,
		ProxyHostUpstream: s.ProxyHostUpstream,
		TLSMin13Only:      s.TLSMin13Only,
		SecurityHeaders:   s.SecurityHeaders,
	}
	if s.IPBlacklistText != nil {
		in.IPBlacklist = proxy.ParseIPListText(*s.IPBlacklistText)
	}
	if s.IPWhitelistText != nil {
		in.IPWhitelist = proxy.ParseIPListText(*s.IPWhitelistText)
	}
	if s.BasicAuth != nil {
		in.BasicAuth = &proxy.BasicAuthInput{
			Enabled:  s.BasicAuth.Enabled,
			Username: s.BasicAuth.Username,
			Password: s.BasicAuth.Password,
		}
	}
	return in
}
