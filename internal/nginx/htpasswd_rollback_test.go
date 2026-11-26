package nginx

import (
	"os"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestHtpasswdSnapshotRestore(t *testing.T) {
	cfg := config.Config{DataDir: t.TempDir()}
	rule := proxy.Rule{ID: 1, Enabled: true, Security: proxy.SecurityConfig{
		BasicAuth: &proxy.BasicAuthConfig{Enabled: true, Username: "first", PasswordHash: "hash"},
	}}
	if err := SyncHtpasswdFiles(cfg, []proxy.Rule{rule}); err != nil { t.Fatal(err) }
	old, err := snapshotHtpasswd(cfg)
	if err != nil { t.Fatal(err) }
	rule.ID = 2
	if err := SyncHtpasswdFiles(cfg, []proxy.Rule{rule}); err != nil { t.Fatal(err) }
	if err := restoreHtpasswd(cfg, old); err != nil { t.Fatal(err) }
	data, err := os.ReadFile(HtpasswdPath(cfg, 1))
	if err != nil || string(data) != "first:hash\n" { t.Fatalf("old auth not restored: %v", err) }
	if _, err := os.Stat(HtpasswdPath(cfg, 2)); !os.IsNotExist(err) { t.Fatal("new auth not removed") }
}
