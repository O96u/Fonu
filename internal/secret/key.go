package secret

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const LegacyDefaultKey = "change-me-in-production"
const KeyFileName = ".secret_key"

// ResolveKey preserves legacy databases rather than silently invalidating credentials.
func ResolveKey(dir, explicit string) (string, error) {
	if explicit != "" && explicit != LegacyDefaultKey { return explicit, nil }
	path := filepath.Join(dir, KeyFileName)
	if raw, err := os.ReadFile(path); err == nil {
		key := strings.TrimSpace(string(raw))
		return key, ValidateStoredKey(key)
	} else if !os.IsNotExist(err) { return "", err }
	key := LegacyDefaultKey
	if _, err := os.Stat(filepath.Join(dir, "fonu.db")); os.IsNotExist(err) {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil { return "", err }
		key = base64.RawStdEncoding.EncodeToString(raw)
	} else if err != nil { return "", err }
	if err := SaveStoredKey(dir, key); err != nil { return "", err }
	return key, nil
}

func ValidateStoredKey(key string) error {
	if key == LegacyDefaultKey { return nil }
	raw, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil || len(raw) != 32 { return fmt.Errorf("invalid stored encryption key") }
	return nil
}

func SaveStoredKey(dir, key string) error {
	if err := ValidateStoredKey(key); err != nil { return err }
	f, err := os.CreateTemp(dir, ".secret-key-*")
	if err != nil { return err }
	defer os.Remove(f.Name())
	_, err = f.WriteString(key)
	if err == nil { err = f.Sync() }
	closeErr := f.Close()
	if err == nil { err = closeErr }
	if err != nil { return err }
	return os.Rename(f.Name(), filepath.Join(dir, KeyFileName))
}
