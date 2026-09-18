package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func Generate(cfg config.Config, rules []proxy.Rule, certs []CertSource, opts GenerateOptions) (string, error) {
	var b strings.Builder

	b.WriteString(`worker_processes auto;
error_log ` + absNginxPath(filepath.Join(cfg.LogsDir(), "error.log")) + ` warn;
pid ` + absNginxPath(cfg.NginxPIDFile) + `;

events {
    worker_connections 1024;
}

http {
    include       ` + absNginxPath(cfg.NginxMimeTypes) + `;
    default_type  application/octet-stream;

    log_format fonu_access '$time_iso8601 $host $server_port $request_method $request_uri $status $request_time $remote_addr $upstream_addr $request_length $bytes_sent';
    access_log ` + absNginxPath(filepath.Join(cfg.LogsDir(), "access.log")) + ` fonu_access;

    sendfile on;
    keepalive_timeout 65;

    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }

    limit_req_status 429;
    limit_conn_status 503;

`)

	writeRealIPDirectives(&b, opts.TrustedProxy)
	writeGlobalAllowList(&b, opts.GlobalIPWhitelist)
	writeGlobalDenyList(&b, opts.GlobalIPBlacklist)
	writeLimitZones(&b, rules)
	writeChinaGeoBlocks(&b, cfg, opts, rules)
	writeChinaOnlyBypassGeo(&b, rules, opts)
	writeGlobalCustomInclude(&b, cfg, opts)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.NginxMode == "custom" {
			if err := writeCustomRuleBlocks(&b, cfg, rule.ID, opts); err != nil {
				return "", err
			}
			continue
		}
		if err := writeRuleBlocks(&b, cfg, rule, certs, opts); err != nil {
			return "", err
		}
	}

	b.WriteString("}\n")
	content := b.String()
	if err := assertValidSSLBlocks(content); err != nil {
		return "", err
	}
	return content, nil
}

func GenerateRuleBlocks(cfg config.Config, rule proxy.Rule, certs []CertSource, opts GenerateOptions) (string, error) {
	var b strings.Builder
	writeRulePreambleComments(&b, rule)
	if err := writeRuleBlocks(&b, cfg, rule, certs, opts); err != nil {
		return "", err
	}
	content := b.String()
	if err := assertValidSSLBlocks(content); err != nil {
		return "", err
	}
	return content, nil
}

func writeGlobalCustomInclude(b *strings.Builder, cfg config.Config, opts GenerateOptions) {
	if strings.TrimSpace(opts.GlobalCustomOverride) != "" {
		writeIndentedHTTPSnippet(b, opts.GlobalCustomOverride)
		return
	}
	if globalCustomShouldInclude(cfg, opts) {
		path := globalCustomPath(cfg, opts)
		b.WriteString(fmt.Sprintf("    include %s;\n\n", absNginxPath(path)))
		return
	}
	writeIndentedHTTPSnippet(b, DefaultGlobalHTTPSnippet())
}

func writeIndentedHTTPSnippet(b *strings.Builder, content string) {
	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		b.WriteString("    " + line + "\n")
	}
	b.WriteString("\n")
}

func writeCustomRuleBlocks(b *strings.Builder, cfg config.Config, ruleID int64, opts GenerateOptions) error {
	content := strings.TrimSpace(ruleCustomContent(cfg, ruleID, opts))
	if content == "" {
		return fmt.Errorf("规则 %d 的自定义 Nginx 配置为空", ruleID)
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	b.WriteString(content)
	return nil
}

func writeRuleBlocks(b *strings.Builder, cfg config.Config, rule proxy.Rule, certs []CertSource, opts GenerateOptions) error {
	for _, group := range rule.PortGroups() {
		if len(group.Hostnames) == 0 {
			continue
		}
		serverNames := strings.Join(group.Hostnames, " ")
		cert := findCertificateForHosts(cfg.CertsDir(), group.Hostnames, certs)
		useHTTPS := rule.HTTPSEnabled && certUsable(cert)

		wroteBlock := false
		if useHTTPS {
			if !appendSSLServerBlock(b, cfg, group.Port, serverNames, cert, rule, opts) {
				return fmt.Errorf("HTTPS 证书文件不可用，请重新导入证书或关闭 HTTPS")
			}
			wroteBlock = true
		}
		if rule.HTTPRedirect && useHTTPS {
			writeServerBlockHeaderComment(b, rule, serverNames, group.Port, "redirect")
			b.WriteString("server {\n")
			writeListenComments(b, group.Port, false, rule.ListenIPv4, rule.ListenIPv6)
			writeListenDirectives(b, group.Port, false, rule.ListenIPv4, rule.ListenIPv6)
			b.WriteString(fmt.Sprintf("    server_name %s;\n", serverNames))
			writeErrorPages(b, cfg)
			writeCommentLine(b, 4, "301 跳转至 HTTPS")
			b.WriteString("    return 301 https://$host:$server_port$request_uri;\n")
			b.WriteString("}\n")
			wroteBlock = true
		}

		if !wroteBlock {
			writeServerBlockHeaderComment(b, rule, serverNames, group.Port, "http")
			b.WriteString("server {\n")
			writeListenComments(b, group.Port, false, rule.ListenIPv4, rule.ListenIPv6)
			writeListenDirectives(b, group.Port, false, rule.ListenIPv4, rule.ListenIPv6)
			b.WriteString(fmt.Sprintf("    server_name %s;\n", serverNames))
			writeErrorPages(b, cfg)
			writeLocationWithSecurity(b, cfg, rule, opts)
			b.WriteString("}\n")
		}
	}
	return nil
}

func appendSSLServerBlock(b *strings.Builder, cfg config.Config, port int, serverNames string, cert *certFiles, rule proxy.Rule, opts GenerateOptions) bool {
	if !certUsable(cert) {
		return false
	}
	var block strings.Builder
	writeServerBlockHeaderComment(&block, rule, serverNames, port, "https")
	block.WriteString("server {\n")
	writeListenComments(&block, port, true, rule.ListenIPv4, rule.ListenIPv6)
	writeListenDirectives(&block, port, true, rule.ListenIPv4, rule.ListenIPv6)
	block.WriteString(fmt.Sprintf("    server_name %s;\n\n", serverNames))
	writeCommentLine(&block, 4, "SSL 证书")
	block.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", absNginxPath(cert.CertPath)))
	block.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", absNginxPath(cert.KeyPath)))
	writeTLSProtocols(&block, rule.Security)
	writeServerSecurityHeaders(&block, rule.Security, true)
	writeErrorPages(&block, cfg)
	writeLocationWithSecurity(&block, cfg, rule, opts)
	block.WriteString("}\n")
	b.WriteString(block.String())
	return true
}

func writeLocationWithSecurity(b *strings.Builder, cfg config.Config, rule proxy.Rule, opts GenerateOptions) {
	writeLocationHeaderComment(b, rule.Upstream)
	b.WriteString("    location / {\n")
	writeLocationSecurity(b, cfg, rule, opts)
	writeProxyLocationBody(b, rule.Upstream, rule.Security)
	b.WriteString("    }\n")
}

func writeProxyLocationBody(b *strings.Builder, upstream string, sec proxy.SecurityConfig) {
	sec = sec.Normalize()
	writeProxyBodyComments(b, upstream, sec)
	hostHeader := "$host"
	if sec.ProxyHostUpstream {
		hostHeader = "$proxy_host"
	}
	sslBlock := ""
	if strings.HasPrefix(upstream, "https://") {
		if sec.ProxySSLVerifyOff {
			sslBlock = `        proxy_ssl_verify off;
        proxy_ssl_server_name on;
`
		} else {
			sslBlock = `        proxy_ssl_server_name on;
`
		}
	}
	b.WriteString(fmt.Sprintf(`%s        proxy_pass %s;
        proxy_http_version 1.1;
        proxy_set_header Host %s;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_set_header X-Real-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
        proxy_read_timeout 3600s;
`, sslBlock, upstream, hostHeader))
}

func writeListenDirectives(b *strings.Builder, port int, ssl bool, ipv4 bool, ipv6 bool) {
	sslSuffix := ""
	if ssl {
		sslSuffix = " ssl"
	}
	if ipv4 {
		b.WriteString(fmt.Sprintf("    listen %d%s;\n", port, sslSuffix))
	}
	if ipv6 {
		b.WriteString(fmt.Sprintf("    listen [::]:%d%s;\n", port, sslSuffix))
	}
}

type certFiles struct {
	CertPath string
	KeyPath  string
}

func findCertificate(certsDir, domain string) *certFiles {
	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts)-1; i++ {
		wildcard := "*." + strings.Join(parts[i:], ".")
		if c := certAt(certsDir, wildcard); c != nil {
			return c
		}
	}
	if c := certAt(certsDir, domain); c != nil {
		return c
	}
	return nil
}

func certAt(certsDir, name string) *certFiles {
	safeName := strings.ReplaceAll(name, "*", "wildcard")
	dir := filepath.Join(certsDir, safeName)
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privatekey.pem")
	if fileExists(certPath) && fileExists(keyPath) {
		return &certFiles{CertPath: absNginxPath(certPath), KeyPath: absNginxPath(keyPath)}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
