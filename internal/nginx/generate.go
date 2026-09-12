package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func Generate(cfg config.Config, rules []proxy.Rule) (string, error) {
	var b strings.Builder

	b.WriteString(`worker_processes auto;
error_log ` + filepath.ToSlash(cfg.LogsDir()) + `/error.log warn;
pid ` + filepath.ToSlash(cfg.NginxPIDFile) + `;

events {
    worker_connections 1024;
}

http {
    include       ` + filepath.ToSlash(cfg.NginxMimeTypes) + `;
    default_type  application/octet-stream;

    log_format fonu_access '$time_iso8601 $host $request_method $request_uri $status $request_time $remote_addr $upstream_addr';
    access_log ` + filepath.ToSlash(cfg.LogsDir()) + `/access.log fonu_access;

    sendfile on;
    keepalive_timeout 65;

    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }

`)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if err := writeRuleBlocks(&b, cfg, rule); err != nil {
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

func writeRuleBlocks(b *strings.Builder, cfg config.Config, rule proxy.Rule) error {
	for _, group := range rule.PortGroups() {
		serverNames := strings.Join(group.Hostnames, " ")
		cert := findCertificateForHosts(cfg.CertsDir(), group.Hostnames)
		useHTTPS := rule.HTTPSEnabled && certUsable(cert)

		if rule.HTTPRedirect && useHTTPS {
			b.WriteString("server {\n")
			writeListenDirectives(b, group.Port, false, rule.ListenIPv4, rule.ListenIPv6)
			b.WriteString(fmt.Sprintf("    server_name %s;\n", serverNames))
			b.WriteString("    return 301 https://$host:$server_port$request_uri;\n")
			b.WriteString("}\n")
		} else if !useHTTPS {
			b.WriteString("server {\n")
			writeListenDirectives(b, group.Port, false, rule.ListenIPv4, rule.ListenIPv6)
			b.WriteString(fmt.Sprintf("    server_name %s;\n", serverNames))
			writeProxyLocation(b, rule.Upstream)
			b.WriteString("}\n")
		}

		if useHTTPS {
			writeSSLServerBlock(b, group.Port, serverNames, cert, rule.Upstream, rule.ListenIPv4, rule.ListenIPv6)
		}
	}
	return nil
}

func writeSSLServerBlock(b *strings.Builder, port int, serverNames string, cert *certFiles, upstream string, ipv4 bool, ipv6 bool) {
	if !certUsable(cert) {
		return
	}
	b.WriteString("server {\n")
	writeListenDirectives(b, port, true, ipv4, ipv6)
	b.WriteString(fmt.Sprintf("    server_name %s;\n\n", serverNames))
	b.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", filepath.ToSlash(cert.CertPath)))
	b.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", filepath.ToSlash(cert.KeyPath)))
	b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n\n")
	writeProxyLocation(b, upstream)
	b.WriteString("}\n")
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

func writeProxyLocation(b *strings.Builder, upstream string) {
	b.WriteString(fmt.Sprintf(`    location / {
        proxy_pass %s;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
        proxy_read_timeout 3600s;
    }
`, upstream))
}

type certFiles struct {
	CertPath string
	KeyPath  string
}

func findCertificateForHosts(certsDir string, hostnames []string) *certFiles {
	for _, hostname := range hostnames {
		if cert := findCertificate(certsDir, hostname); cert != nil {
			return cert
		}
	}
	return nil
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
		return &certFiles{CertPath: certPath, KeyPath: keyPath}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
