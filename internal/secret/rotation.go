package secret

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const rotationFile = ".secret_key.next"

func HasPendingRotation(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, rotationFile))
	return !os.IsNotExist(err)
}

// A durable pending key is written before touching the database. Every value
// is accepted under either key, making a restart after commit safe.
func RotateLegacyKey(ctx context.Context, db *sql.DB, dir, current string) (string, error) {
	path := filepath.Join(dir, rotationFile)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if current != LegacyDefaultKey { return current, nil }
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil { return "", err }
		raw = []byte(base64.RawStdEncoding.EncodeToString(b))
		if err := writeKeyFile(dir, rotationFile, string(raw)); err != nil { return "", err }
	} else if err != nil { return "", err }
	next := string(raw)
	if next == LegacyDefaultKey { return "", fmt.Errorf("invalid rotation key") }
	if err := ValidateStoredKey(next); err != nil { return "", err }
	oldBox, err := NewBox(LegacyDefaultKey)
	if err != nil { return "", err }
	newBox, err := NewBox(next)
	if err != nil { return "", err }
	convert := func(value string) (string, error) {
		if value == "" { return "", nil }
		if _, err := newBox.Decrypt(value); err == nil { return value, nil }
		plain, err := oldBox.Decrypt(value)
		if err != nil { return "", err }
		return newBox.Encrypt(plain)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil { return "", err }
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id, api_token_enc FROM ddns_configs WHERE api_token_enc IS NOT NULL AND api_token_enc != ''")
	if err != nil { return "", err }
	type item struct { id int64; value string }
	var items []item
	for rows.Next() {
		var i item
		if err := rows.Scan(&i.id, &i.value); err != nil { rows.Close(); return "", err }
		items = append(items, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil { return "", err }
	for _, i := range items {
		v, err := convert(i.value)
		if err != nil { return "", fmt.Errorf("rotate DDNS credential %d: %w", i.id, err) }
		if _, err := tx.ExecContext(ctx, "UPDATE ddns_configs SET api_token_enc = ? WHERE id = ?", v, i.id); err != nil { return "", err }
	}
	for _, key := range []string{"frp_auth_token", "notify_smtp_password", "notify_webhook_secret", "notify_telegram_token", "dnshe_api_key", "dnshe_api_secret", "dnshe_accounts"} {
		var value string
		err := tx.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) { continue }
		if err != nil { return "", err }
		if value == "" { continue }
		var updated string
		if key == "dnshe_accounts" {
			var accounts []map[string]json.RawMessage
			if err := json.Unmarshal([]byte(value), &accounts); err != nil { return "", err }
			for _, account := range accounts {
				for _, field := range []string{"api_key_enc", "api_secret_enc"} {
					raw, ok := account[field]; if !ok { continue }
					var enc string
					if err := json.Unmarshal(raw, &enc); err != nil { return "", err }
					v, err := convert(enc); if err != nil { return "", fmt.Errorf("rotate DNSHE %s: %w", field, err) }
					account[field], err = json.Marshal(v); if err != nil { return "", err }
				}
			}
			raw, err := json.Marshal(accounts); if err != nil { return "", err }; updated = string(raw)
		} else {
			updated, err = convert(value); if err != nil { return "", fmt.Errorf("rotate %s: %w", key, err) }
		}
		if _, err := tx.ExecContext(ctx, "UPDATE settings SET value = ? WHERE key = ?", updated, key); err != nil { return "", err }
	}
	if err := tx.Commit(); err != nil { return "", err }
	if err := SaveStoredKey(dir, next); err != nil { return "", err }
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) { return "", err }
	return next, nil
}
