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
	return b.String(), nil
}

func writeRuleBlocks(b *strings.Builder, cfg config.Config, rule proxy.Rule) error {
	cert := findCertificate(cfg.CertsDir(), rule.Domain)
	hasCert := cert != nil

	if rule.HTTPRedirect && rule.HTTPSEnabled && hasCert {
		b.WriteString(fmt.Sprintf(`
server {
    listen 80;
    server_name %s;
    return 301 https://$host$request_uri;
}
`, rule.Domain))
	} else {
		b.WriteString(fmt.Sprintf(`
server {
    listen 80;
    server_name %s;
`, rule.Domain))
		writeProxyLocation(b, rule.Upstream)
		b.WriteString("}\n")
	}

	if rule.HTTPSEnabled && hasCert {
		b.WriteString(fmt.Sprintf(`
server {
    listen 443 ssl;
    server_name %s;

    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;

`, rule.Domain, filepath.ToSlash(cert.CertPath), filepath.ToSlash(cert.KeyPath)))
		writeProxyLocation(b, rule.Upstream)
		b.WriteString("}\n")
	}

	return nil
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
