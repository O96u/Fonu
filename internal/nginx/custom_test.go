package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateRuleBlocks(t *testing.T) {
	cfg := config.Config{DataDir: t.TempDir()}
	rule := proxy.Rule{
		ID:           1,
		Upstream:     "http://127.0.0.1:8080",
		ListenPort:   8011,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "example.com"}},
		HTTPSEnabled: false,
		Enabled:      true,
	}
	out, err := GenerateRuleBlocks(cfg, rule, nil, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "server_name example.com") {
		t.Fatalf("missing server_name: %s", out)
	}
	if !strings.Contains(out, "proxy_pass http://127.0.0.1:8080") {
		t.Fatalf("missing proxy_pass: %s", out)
	}
}

func TestGenerateWithCustomRuleAndGlobal(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DataDir: dir}
	if err := EnsureCustomDirs(cfg); err != nil {
		t.Fatal(err)
	}
	customGlobal := "gzip on;\n"
	if err := WriteGlobalCustom(cfg, customGlobal); err != nil {
		t.Fatal(err)
	}
	customRule := "server {\n    listen 8011;\n    server_name custom.test;\n    return 444;\n}\n"
	if err := WriteRuleCustom(cfg, 7, customRule); err != nil {
		t.Fatal(err)
	}

	rules := []proxy.Rule{{
		ID:         7,
		Upstream:   "http://127.0.0.1:1",
		ListenPort: 8011,
		ListenIPv4: true,
		Hosts:      []proxy.Host{{Hostname: "ignored.test"}},
		Enabled:    true,
		NginxMode:  "custom",
	}}
	out, err := Generate(cfg, rules, nil, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "include "+strings.ReplaceAll(absNginxPath(cfg.NginxGlobalCustomPath()), "\\", "/")) &&
		!strings.Contains(out, absNginxPath(cfg.NginxGlobalCustomPath())) {
		t.Fatalf("missing global include: %s", out)
	}
	if !strings.Contains(out, "server_name custom.test") {
		t.Fatalf("missing custom rule block: %s", out)
	}
	if strings.Contains(out, "ignored.test") {
		t.Fatalf("auto-generated host leaked into custom mode: %s", out)
	}
}

func TestBackupFileKeepsRecent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.conf")
	if err := os.WriteFile(src, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(dir, "backups")
	if _, err := BackupFile(src, backupDir); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := os.WriteFile(src, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := BackupFile(src, backupDir); err != nil {
		t.Fatal(err)
	}
	entries, err := ListBackups(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("backups=%d want 2", len(entries))
	}
	if err := RestoreBackup(backupDir, entries[1].Name, src); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "v1" {
		t.Fatalf("restored=%q want v1", string(data))
	}
}

func TestSaveRuleCustomWithBackup(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DataDir: dir}
	if err := WriteRuleCustom(cfg, 3, "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveRuleCustomWithBackup(cfg, 3, "second"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cfg.NginxRuleCustomPath(3))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "second" {
		t.Fatalf("content=%q", string(data))
	}
	backups, err := ListBackups(cfg.NginxRuleBackupsDir(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups=%d want 1", len(backups))
	}
}
