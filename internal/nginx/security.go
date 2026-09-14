package nginx

import (
	"fmt"
	"os"
	"strings"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

type GenerateOptions struct {
	TrustedProxy         TrustedProxyConfig
	GlobalIPBlacklist    []string
	ChinaCIDRAvailable   bool
	ChinaCIDRPathOverride string
}

func writeGlobalDenyList(b *strings.Builder, cidrs []string) {
	for _, cidr := range cidrs {
		b.WriteString(fmt.Sprintf("    deny %s;\n", cidr))
	}
	if len(cidrs) > 0 {
		b.WriteString("\n")
	}
}

func writeLimitZones(b *strings.Builder, rules []proxy.Rule) {
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.Security.RateLimitEnabled() {
			rl := rule.Security.RateLimit
			b.WriteString(fmt.Sprintf("    limit_req_zone $binary_remote_addr zone=fonu_rule_%d_req:10m rate=%dr/s;\n", rule.ID, rl.Rate))
		}
		if rule.Security.ConnLimitEnabled() {
			b.WriteString(fmt.Sprintf("    limit_conn_zone $binary_remote_addr zone=fonu_rule_%d_conn:10m;\n", rule.ID))
		}
	}
	if hasLimitZones(rules) {
		b.WriteString("\n")
	}
}

func hasLimitZones(rules []proxy.Rule) bool {
	for _, rule := range rules {
		if rule.Enabled && (rule.Security.RateLimitEnabled() || rule.Security.ConnLimitEnabled()) {
			return true
		}
	}
	return false
}

func writeChinaGeoBlocks(b *strings.Builder, cfg config.Config, opts GenerateOptions, rules []proxy.Rule) {
	if !opts.ChinaCIDRAvailable {
		return
	}
	needsChina := false
	for _, rule := range rules {
		if rule.Enabled && rule.Security.ChinaOnly {
			needsChina = true
			break
		}
	}
	if !needsChina {
		return
	}
	cidrPath := absNginxPath(cfg.ChinaCIDRPath())
	if opts.ChinaCIDRPathOverride != "" {
		cidrPath = absNginxPath(opts.ChinaCIDRPathOverride)
	}
	b.WriteString(fmt.Sprintf(`    geo $fonu_client_ip $fonu_is_china {
        default 0;
        include %s;
    }

`, cidrPath))
}

func writeChinaOnlyBypassGeo(b *strings.Builder, rules []proxy.Rule, opts GenerateOptions) {
	if !opts.ChinaCIDRAvailable {
		return
	}
	for _, rule := range rules {
		if !rule.Enabled || !rule.Security.ChinaOnly {
			continue
		}
		sec := rule.Security.Normalize()
		if len(sec.IPWhitelist) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("    geo $fonu_client_ip $fonu_rule_%d_china_bypass {\n", rule.ID))
		b.WriteString("        default 0;\n")
		for _, cidr := range sec.IPWhitelist {
			b.WriteString(fmt.Sprintf("        %s 1;\n", cidr))
		}
		b.WriteString("    }\n\n")
	}
}

func writeChinaOnlyCheck(b *strings.Builder, rule proxy.Rule, opts GenerateOptions) {
	sec := rule.Security.Normalize()
	if !sec.ChinaOnly || !opts.ChinaCIDRAvailable {
		return
	}
	if len(sec.IPWhitelist) == 0 {
		b.WriteString(`        if ($fonu_is_china = 0) {
            return 403;
        }
`)
		return
	}
	b.WriteString(`        set $fonu_china_block 0;
        if ($fonu_is_china = 0) {
            set $fonu_china_block 1;
        }
`)
	b.WriteString(fmt.Sprintf(`        if ($fonu_rule_%d_china_bypass = 1) {
            set $fonu_china_block 0;
        }
        if ($fonu_china_block = 1) {
            return 403;
        }
`, rule.ID))
}

func writeServerSecurityHeaders(b *strings.Builder, sec proxy.SecurityConfig, https bool) {
	if !https || !sec.SecurityHeaders {
		return
	}
	b.WriteString(`    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;

`)
}

func writeTLSProtocols(b *strings.Builder, sec proxy.SecurityConfig) {
	if sec.TLSMin13Only {
		b.WriteString("    ssl_protocols TLSv1.3;\n\n")
	} else {
		b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n\n")
	}
}

func writeErrorPages(b *strings.Builder, cfg config.Config) {
	errorsDir := absNginxPath(cfg.ErrorsDir())
	b.WriteString(`    error_page 403 /fonu-errors/403.html;
    error_page 404 /fonu-errors/404.html;
    error_page 429 /fonu-errors/429.html;
    error_page 500 /fonu-errors/500.html;
    error_page 502 503 /fonu-errors/503.html;

    location ^~ /fonu-errors/ {
        internal;
        alias ` + errorsDir + `/;
    }

`)
}

func writeLocationSecurity(b *strings.Builder, cfg config.Config, rule proxy.Rule, opts GenerateOptions) {
	sec := rule.Security.Normalize()

	for _, cidr := range sec.IPBlacklist {
		b.WriteString(fmt.Sprintf("        deny %s;\n", cidr))
	}

	writeChinaOnlyCheck(b, rule, opts)

	if sec.WhitelistEnabled() {
		for _, cidr := range sec.IPWhitelist {
			b.WriteString(fmt.Sprintf("        allow %s;\n", cidr))
		}
		b.WriteString("        deny all;\n")
	}

	if sec.ConnLimitEnabled() {
		b.WriteString(fmt.Sprintf("        limit_conn fonu_rule_%d_conn %d;\n", rule.ID, sec.ConnLimit.Max))
	}
	if sec.RateLimitEnabled() {
		b.WriteString(fmt.Sprintf("        limit_req zone=fonu_rule_%d_req burst=%d nodelay;\n", rule.ID, sec.RateLimit.Burst))
	}

	if sec.BasicAuthEnabled() {
		path := absNginxPath(HtpasswdPath(cfg, rule.ID))
		b.WriteString(fmt.Sprintf(`        auth_basic "Restricted";
        auth_basic_user_file %s;
`, path))
	}
}

func chinaCIDRExists(cfg config.Config) bool {
	info, err := os.Stat(cfg.ChinaCIDRPath())
	return err == nil && !info.IsDir() && info.Size() > 0
}
