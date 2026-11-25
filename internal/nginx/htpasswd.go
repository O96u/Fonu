package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/validate"
)

// nginxHtpasswdHash normalizes bcrypt hashes for nginx/apr auth_basic.
// Go's bcrypt uses the $2a$ prefix; some nginx builds only accept $2y$.
func nginxHtpasswdHash(hash string) string {
	if strings.HasPrefix(hash, "$2a$") {
		return "$2y$" + hash[4:]
	}
	return hash
}

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
	ensureNginxReadable(dir, true)

	active := map[int64]bool{}
	for _, rule := range rules {
		path := HtpasswdPath(cfg, rule.ID)
		if rule.Security.BasicAuthEnabled() {
			if err := validate.BasicAuthUsername(rule.Security.BasicAuth.Username); err != nil { return err }
			active[rule.ID] = true
			line := fmt.Sprintf("%s:%s\n", rule.Security.BasicAuth.Username, nginxHtpasswdHash(rule.Security.BasicAuth.PasswordHash))
			if err := writeHtpasswdAtomic(path, []byte(line)); err != nil {
				return err
			}
			ensureNginxReadable(path, false)
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
		if entry.IsDir() || !managedHtpasswd(entry.Name()) {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(entry.Name(), "rule_%d.conf", &id); err != nil {
			continue
		}
		if !active[id] {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) { return err }
		}
	}
	return nil
}

// Only manage files with exactly the generated rule filename format.
func managedHtpasswd(name string) bool {
	var id int64
	_, err := fmt.Sscanf(name, "rule_%d.conf", &id)
	return err == nil && id > 0 && name == fmt.Sprintf("rule_%d.conf", id)
}

func snapshotHtpasswd(cfg config.Config) (map[string][]byte, error) {
	out := make(map[string][]byte)
	dir := HtpasswdDir(cfg)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) { return out, nil }
	if err != nil { return nil, err }
	for _, entry := range entries {
		if !managedHtpasswd(entry.Name()) { continue }
		if !entry.Type().IsRegular() { return nil, fmt.Errorf("invalid htpasswd file: %s", entry.Name()) }
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil { return nil, err }
		out[entry.Name()] = data
	}
	return out, nil
}

func restoreHtpasswd(cfg config.Config, snapshot map[string][]byte) error {
	dir := HtpasswdDir(cfg)
	if err := os.MkdirAll(dir, 0755); err != nil { return err }
	for name, data := range snapshot {
		if err := writeHtpasswdAtomic(filepath.Join(dir, name), data); err != nil { return err }
	}
	entries, err := os.ReadDir(dir)
	if err != nil { return err }
	for _, entry := range entries {
		if !managedHtpasswd(entry.Name()) { continue }
		if _, ok := snapshot[entry.Name()]; !ok {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil { return err }
		}
	}
	return nil
}

func writeHtpasswdAtomic(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".htpasswd-*")
	if err != nil { return err }
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	if err == nil { err = f.Sync() }
	closeErr := f.Close()
	if err == nil { err = closeErr }
	if err != nil { return err }
	ensureNginxReadable(f.Name(), false)
	return os.Rename(f.Name(), path)
}
