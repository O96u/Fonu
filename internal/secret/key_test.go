package secret

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKey(t *testing.T) {
	dir := t.TempDir()
	key, err := ResolveKey(dir, "")
	if err != nil || key == LegacyDefaultKey { t.Fatalf("fresh key: %v", err) }
	again, err := ResolveKey(dir, "")
	if err != nil || again != key { t.Fatal("key was not persisted") }
	explicit, err := ResolveKey(dir, "custom-key")
	if err != nil || explicit != "custom-key" { t.Fatal("explicit key ignored") }
	legacy := t.TempDir()
	if err := os.WriteFile(filepath.Join(legacy, "fonu.db"), nil, 0600); err != nil { t.Fatal(err) }
	old, err := ResolveKey(legacy, "")
	if err != nil || old != LegacyDefaultKey { t.Fatal("legacy key not preserved") }
}
