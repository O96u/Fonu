package nginx

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestSyncHtpasswdFilesWritesBasicAuth(t *testing.T) {
	cfg := config.Config{DataDir: t.TempDir()}
	rules := []proxy.Rule{{
		ID:      3,
		Enabled: true,
		Security: proxy.SecurityConfig{
			BasicAuth: &proxy.BasicAuthConfig{
				Enabled:      true,
				Username:     "admin",
				PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwx.yz012345678901234567890",
			},
		},
	}}

	if err := SyncHtpasswdFiles(cfg, rules); err != nil {
		t.Fatalf("sync htpasswd: %v", err)
	}

	path := HtpasswdPath(cfg, 3)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read htpasswd: %v", err)
	}
	if !strings.HasPrefix(string(data), "admin:$2y$10$") {
		t.Fatalf("unexpected htpasswd content: %q", data)
	}
	if runtime.GOOS == "linux" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat htpasswd: %v", err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("expected htpasswd mode 0644, got %o", info.Mode().Perm())
		}
	}

	stripped := rules[0].Security.ForAPI()
	rules[0].Security = stripped
	if rules[0].Security.BasicAuthEnabled() {
		t.Fatal("ForAPI should strip password hash for API responses")
	}
	if err := SyncHtpasswdFiles(cfg, rules); err != nil {
		t.Fatalf("sync stripped htpasswd: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected htpasswd removed when hash stripped, stat err=%v", err)
	}
}
