package nginx

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func HtpasswdDir(cfg config.Config) string {
	return filepath.Join(cfg.NginxDir(), "htpasswd")
}

func HtpasswdPath(cfg config.Config, ruleID int64) string {
	return filepath.Join(HtpasswdDir(cfg), fmt.Sprintf("rule_%d.conf", ruleID))
}

func SyncHtpasswdFiles(cfg config.Config, rules []proxy.Rule) error {
	dir := HtpasswdDir(cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	active := map[int64]bool{}
	for _, rule := range rules {
		path := HtpasswdPath(cfg, rule.ID)
		if rule.Security.BasicAuthEnabled() {
			active[rule.ID] = true
			line := fmt.Sprintf("%s:%s\n", rule.Security.BasicAuth.Username, rule.Security.BasicAuth.PasswordHash)
			if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
				return err
			}
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(entry.Name(), "rule_%d.conf", &id); err != nil {
			continue
		}
		if !active[id] {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}
