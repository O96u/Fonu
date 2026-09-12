package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/fonu/fonu/internal/app"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
)

func main() {
	cfg := config.Load()
	migrationsDir, err := app.ResolveMigrationsDir()
	if err != nil {
		log.Fatalf("migrations: %v", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("mkdir data: %v", err)
	}
	for _, dir := range []string{cfg.LogsDir(), cfg.CertsDir(), cfg.NginxDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	conn, err := db.Open(cfg.DBPath())
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx, conn, migrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := seedAdmin(ctx, conn); err != nil {
		log.Fatalf("seed admin: %v", err)
	}
	if err := seedSettings(ctx, conn); err != nil {
		log.Fatalf("seed settings: %v", err)
	}
	if err := seedProxies(ctx, conn); err != nil {
		log.Fatalf("seed proxies: %v", err)
	}
	if err := seedDDNS(ctx, conn, cfg); err != nil {
		log.Fatalf("seed ddns: %v", err)
	}
	if err := seedCertificates(ctx, conn, cfg); err != nil {
		log.Fatalf("seed certificates: %v", err)
	}
	if err := seedLogs(cfg); err != nil {
		log.Fatalf("seed logs: %v", err)
	}

	fmt.Println()
	fmt.Println("本地模拟数据已写入：", cfg.DataDir)
	fmt.Println()
	fmt.Println("登录账号：admin / password123")
	fmt.Println("DDNS 域名：example.com、example.org（模拟）")
	fmt.Println()
	fmt.Println("已包含：")
	fmt.Println("  - 5 条反向代理规则")
	fmt.Println("  - DDNS 配置（Cloudflare Token 为假数据）")
	fmt.Println("  - 1 条证书记录")
	fmt.Println("  - 系统 / 访问 / 错误日志样本")
	fmt.Println()
	fmt.Println("启动：")
	fmt.Println("  $env:FONU_DATA_DIR = \"" + cfg.DataDir + "\"")
	fmt.Println("  $env:FONU_SESSION_SECRET = \"dev-secret-change-me\"")
	fmt.Println("  go run ./cmd/fonu")
}

func seedAdmin(ctx context.Context, conn *sql.DB) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(1) FROM admins`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		log.Println("admin: 已存在，跳过（沿用现有密码）")
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO admins(username, password_hash) VALUES (?, ?)`, "admin", string(hash))
	log.Println("admin: 已创建 admin / password123")
	return err
}

func seedSettings(ctx context.Context, conn *sql.DB) error {
	settings := map[string]string{
		"theme":                       "system",
		"timezone":                    "Asia/Shanghai",
		"root_domain":                 "example.com",
		"acme_email":                  "admin@example.com",
		"ddns_check_interval_minutes": "5",
		"cert_renew_threshold_days":   "30",
		"log_retention_days":          "30",
	}
	for k, v := range settings {
		_, err := conn.ExecContext(ctx, `
			INSERT INTO settings(key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, k, v)
		if err != nil {
			return err
		}
	}
	log.Println("settings: 已写入")
	return nil
}

func seedProxies(ctx context.Context, conn *sql.DB) error {
	rules := []struct {
		domain, upstream       string
		https, redirect, enabled int
	}{
		{"nas.example.com", "http://192.168.1.10:5666", 1, 1, 1},
		{"alist.example.com", "http://192.168.1.10:5244", 1, 1, 1},
		{"jellyfin.example.com", "http://192.168.1.10:8096", 1, 1, 1},
		{"photos.example.com", "http://192.168.1.20:2342", 1, 0, 1},
		{"dev.example.com", "http://127.0.0.1:3000", 0, 0, 0},
	}
	for _, r := range rules {
		_, err := conn.ExecContext(ctx, `
			INSERT INTO proxy_rules(domain, upstream, https_enabled, http_redirect, enabled, updated_at)
			VALUES (?, ?, ?, ?, ?, datetime('now'))
			ON CONFLICT(domain) DO UPDATE SET
				upstream = excluded.upstream,
				https_enabled = excluded.https_enabled,
				http_redirect = excluded.http_redirect,
				enabled = excluded.enabled,
				updated_at = datetime('now')
		`, r.domain, r.upstream, r.https, r.redirect, r.enabled)
		if err != nil {
			return err
		}
	}
	log.Println("proxy_rules: 已写入 5 条")
	return nil
}

func seedDDNS(ctx context.Context, conn *sql.DB, cfg config.Config) error {
	box, err := secret.NewBox(cfg.SessionSecret)
	if err != nil {
		return err
	}
	tokenEnc, err := box.Encrypt("cf_mock_token_for_local_dev_only")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, `DELETE FROM ddns_configs`)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	domains := []struct {
		provider, rootDomain, ipv4, ipv6 string
	}{
		{"cloudflare", "example.com", "123.45.67.89", ""},
		{"dnspod", "example.org", "123.45.67.89", ""},
	}
	for _, d := range domains {
		_, err = conn.ExecContext(ctx, `
			INSERT INTO ddns_configs(
				provider, root_domain, record_name, ipv4_enabled, ipv6_enabled, enabled,
				api_token_enc, last_ipv4, last_ipv6, last_status, last_updated_at, updated_at
			) VALUES (?, ?, '*', 1, 0, 1, ?, ?, ?, 'ok', ?, datetime('now'))
		`, d.provider, d.rootDomain, tokenEnc, d.ipv4, d.ipv6, now)
		if err != nil {
			return err
		}
	}
	log.Println("ddns_configs: 已写入 2 条（Token 为本地假数据）")
	return nil
}

func seedCertificates(ctx context.Context, conn *sql.DB, cfg config.Config) error {
	expires := time.Now().Add(62 * 24 * time.Hour).UTC().Format(time.RFC3339)
	renewed := time.Now().Add(-7 * 24 * time.Hour).UTC().Format(time.RFC3339)
	certDir := filepath.Join(cfg.CertsDir(), "example.com")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		return err
	}
	certPath := filepath.Join(certDir, "fullchain.pem")
	keyPath := filepath.Join(certDir, "privatekey.pem")
	if err := os.WriteFile(certPath, []byte("-----BEGIN CERTIFICATE-----\nMOCK LOCAL DEV CERT\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, []byte("-----BEGIN PRIVATE KEY-----\nMOCK LOCAL DEV KEY\n-----END PRIVATE KEY-----\n"), 0o600); err != nil {
		return err
	}

	_, err := conn.ExecContext(ctx, `
		INSERT INTO certificates(domain, wildcard, acme_ca, cert_path, key_path, expires_at, last_renew_at, status, updated_at)
		VALUES (?, 1, 'imported', ?, ?, ?, ?, 'ok', datetime('now'))
		ON CONFLICT(domain) DO UPDATE SET
			wildcard = excluded.wildcard,
			acme_ca = excluded.acme_ca,
			cert_path = excluded.cert_path,
			key_path = excluded.key_path,
			expires_at = excluded.expires_at,
			last_renew_at = excluded.last_renew_at,
			status = excluded.status,
			updated_at = datetime('now')
	`, "example.com", certPath, keyPath, expires, renewed)
	if err != nil {
		return err
	}
	log.Println("certificates: 已写入 1 条（占位证书文件）")
	return nil
}

func seedLogs(cfg config.Config) error {
	today := time.Now().Format("2006-01-02")
	access := fmt.Sprintf(`%sT08:12:01+08:00 nas.example.com GET / 200 0.018 123.45.67.89 192.168.1.10:5666
%sT08:15:22+08:00 alist.example.com GET /api/fs/list 200 0.042 111.22.33.44 192.168.1.10:5244
%sT09:01:33+08:00 jellyfin.example.com GET /Videos 200 0.156 98.76.54.32 192.168.1.10:8096
%sT10:20:11+08:00 nas.example.com POST /api/login 401 0.003 45.67.89.10 192.168.1.10:5666
%sT11:05:44+08:00 photos.example.com GET /album/1 200 0.028 123.45.67.89 192.168.1.20:2342
%sT12:30:00+08:00 nas.example.com GET /health 200 0.002 127.0.0.1 192.168.1.10:5666
%sT14:18:09+08:00 alist.example.com GET / 502 1.204 123.45.67.89 192.168.1.10:5244
%sT15:45:17+08:00 nas.example.com GET /dashboard 200 0.021 123.45.67.89 192.168.1.10:5666
`, today, today, today, today, today, today, today, today)

	errorLog := `2026/09/11 14:18:09 [error] 12345#0: *42 connect() failed (111: Connection refused) while connecting to upstream
2026/09/11 09:55:02 [warn] 12345#0: *18 an upstream response is buffered to a temporary file
`

	now := time.Now().UTC().Format(time.RFC3339)
	appLog := fmt.Sprintf(`{"time":"%s","level":"INFO","module":"SYSTEM","msg":"application started","listen":":6893"}
{"time":"%s","level":"INFO","module":"NGINX","msg":"nginx validate succeeded"}
{"time":"%s","level":"INFO","module":"NGINX","msg":"nginx reloaded"}
{"time":"%s","level":"INFO","module":"DDNS","msg":"ddns ipv4 updated","ip":"123.45.67.89"}
{"time":"%s","level":"WARN","module":"ACME","msg":"certificate renew skipped in local dev"}
{"time":"%s","level":"ERROR","module":"NGINX","msg":"upstream connection refused","domain":"alist.example.com"}
`, now, now, now, now, now, now)

	if err := os.WriteFile(filepath.Join(cfg.LogsDir(), "access.log"), []byte(access), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(cfg.LogsDir(), "error.log"), []byte(errorLog), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(cfg.LogsDir(), "app.log"), []byte(appLog), 0o644); err != nil {
		return err
	}
	log.Println("logs: 已写入 access / error / app 样本")
	return nil
}
