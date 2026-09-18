package nginx

import (
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/proxy"
)

func writeCommentLine(b *strings.Builder, indent int, text string) {
	b.WriteString(strings.Repeat(" ", indent) + "# " + text + "\n")
}

func ruleDisplayName(rule proxy.Rule) string {
	name := strings.TrimSpace(rule.Name)
	if name != "" {
		return name
	}
	if host := rule.PrimaryHost(); host != "" {
		return host
	}
	return fmt.Sprintf("规则 #%d", rule.ID)
}

func writeRulePreambleComments(b *strings.Builder, rule proxy.Rule) {
	sec := rule.Security.Normalize()
	writeCommentLine(b, 0, fmt.Sprintf("规则：%s（ID %d）", ruleDisplayName(rule), rule.ID))
	writeCommentLine(b, 0, fmt.Sprintf("上游：%s", rule.Upstream))
	if sec.RateLimitEnabled() || sec.ConnLimitEnabled() {
		writeCommentLine(b, 0, "限流 zone 在 http 块中自动生成，本段 location 内引用")
	}
	b.WriteString("\n")
}

func writeServerBlockHeaderComment(b *strings.Builder, rule proxy.Rule, serverNames string, port int, kind string) {
	writeCommentLine(b, 0, fmt.Sprintf("规则：%s", ruleDisplayName(rule)))
	writeCommentLine(b, 0, fmt.Sprintf("域名：%s（端口 %d）", serverNames, port))
	switch kind {
	case "https":
		writeCommentLine(b, 0, "HTTPS 反代")
	case "redirect":
		writeCommentLine(b, 0, "HTTP 跳转 HTTPS")
	default:
		writeCommentLine(b, 0, "HTTP 反代")
	}
	b.WriteString("\n")
}

func writeListenComments(b *strings.Builder, port int, ssl bool, ipv4 bool, ipv6 bool) {
	proto := "HTTP"
	if ssl {
		proto = "HTTPS"
	}
	if ipv4 {
		writeCommentLine(b, 4, fmt.Sprintf("监听 %s IPv4 :%d", proto, port))
	}
	if ipv6 {
		writeCommentLine(b, 4, fmt.Sprintf("监听 %s IPv6 :%d", proto, port))
	}
}

func writeTLSProtocolComment(b *strings.Builder, sec proxy.SecurityConfig) {
	if sec.TLSMin13Only {
		writeCommentLine(b, 4, "TLS：仅 TLS 1.3")
	} else {
		writeCommentLine(b, 4, "TLS：TLS 1.2 / TLS 1.3")
	}
}

func writeSecurityHeadersComment(b *strings.Builder) {
	writeCommentLine(b, 4, "安全响应头：HSTS / X-Content-Type-Options / X-Frame-Options")
}

func writeErrorPagesComment(b *strings.Builder) {
	writeCommentLine(b, 4, "Fonu 自定义错误页（403/404/429/5xx）")
}

func writeLocationHeaderComment(b *strings.Builder, upstream string) {
	writeCommentLine(b, 4, fmt.Sprintf("反代 location / → %s", upstream))
}

func writeLocationSecurityComments(b *strings.Builder, rule proxy.Rule, opts GenerateOptions) {
	sec := rule.Security.Normalize()

	if len(sec.IPBlacklist) > 0 {
		writeCommentLine(b, 8, "IP 黑名单")
	}
	if sec.ChinaOnly {
		if opts.ChinaCIDRAvailable {
			writeCommentLine(b, 8, "仅中国大陆：内网 IP 与白名单 IP 段自动豁免")
		} else {
			writeCommentLine(b, 8, "仅中国大陆：中国 IP 段未就绪，当前仅允许内网 IP")
		}
	}
	if sec.WhitelistEnabled() {
		writeCommentLine(b, 8, "IP 白名单模式：仅允许以下 IP，其余拒绝")
	}
	if sec.ConnLimitEnabled() {
		writeCommentLine(b, 8, fmt.Sprintf("连接数限制：每 IP 最多 %d 并发", sec.ConnLimit.Max))
	}
	if sec.RateLimitEnabled() {
		writeCommentLine(b, 8, fmt.Sprintf("请求限流：%d 次/秒，突发 %d（超出返回 429）", sec.RateLimit.Rate, sec.RateLimit.Burst))
	}
	if sec.BasicAuthEnabled() {
		writeCommentLine(b, 8, "Basic Auth：浏览器弹窗认证")
	}
}

func writeProxyBodyComments(b *strings.Builder, upstream string, sec proxy.SecurityConfig) {
	writeCommentLine(b, 8, "反向代理与 WebSocket 头")
	if sec.ProxyHostUpstream {
		writeCommentLine(b, 8, "Host 头：使用上游地址（$proxy_host）")
	}
	if strings.HasPrefix(upstream, "https://") {
		if sec.ProxySSLVerifyOff {
			writeCommentLine(b, 8, "上游 HTTPS：跳过证书校验")
		} else {
			writeCommentLine(b, 8, "上游 HTTPS：启用 SNI")
		}
	}
}

func writeLimitZoneComments(b *strings.Builder, rule proxy.Rule) {
	sec := rule.Security.Normalize()
	if sec.RateLimitEnabled() {
		writeCommentLine(b, 4, fmt.Sprintf("规则 %s：请求限流 zone（%d 次/秒）", ruleDisplayName(rule), sec.RateLimit.Rate))
	}
	if sec.ConnLimitEnabled() {
		writeCommentLine(b, 4, fmt.Sprintf("规则 %s：连接数 zone（每 IP %d 并发）", ruleDisplayName(rule), sec.ConnLimit.Max))
	}
}

func writeGlobalDenyListComment(b *strings.Builder) {
	writeCommentLine(b, 4, "全局 IP 黑名单（系统设置，优先于所有规则）")
}

func writeRealIPComment(b *strings.Builder, cfg TrustedProxyConfig) {
	if len(cfg.effectiveCIDRs()) == 0 {
		writeCommentLine(b, 4, "未启用信任代理：客户端 IP 取 $remote_addr")
		return
	}
	writeCommentLine(b, 4, fmt.Sprintf("信任代理：从 %s 解析真实客户端 IP", cfg.effectiveHeader()))
}

func writeChinaGeoComment(b *strings.Builder) {
	writeCommentLine(b, 4, "仅中国大陆：内网 IP 判定与中国 IP 段库")
}

func writeChinaBypassComment(b *strings.Builder, rule proxy.Rule) {
	writeCommentLine(b, 4, fmt.Sprintf("规则 %s：仅中国大陆 — 白名单 IP 豁免段", ruleDisplayName(rule)))
}
