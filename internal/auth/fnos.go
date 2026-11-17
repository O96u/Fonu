package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"unicode"

	"github.com/fonu/fonu/internal/settings"
	"golang.org/x/crypto/bcrypt"
)

const (
	SourcePassword = "password"
	SourceFnos     = "fnos"
)

var ErrFnosNotBound = errors.New("fnos user is not bound")

// FnosBinding is a NAS account an administrator explicitly allowed to sign in.
type FnosBinding struct {
	UID       string `json:"nas_uid"`
	Username  string `json:"nas_username"`
	CreatedAt string `json:"created_at"`
}

func (s *Service) FnosEnabled(ctx context.Context) (bool, error) {
	value, err := s.setting(ctx, settings.KeyFnosAuthEnabled)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	default:
		return false, nil
	}
}

func (s *Service) SetFnosEnabled(ctx context.Context, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	if err := s.setSetting(ctx, settings.KeyFnosAuthEnabled, value); err != nil {
		return err
	}
	if !enabled {
		if err := s.RevokeFnosSessions(ctx, ""); err != nil {
			return err
		}
	}
	slog.Info("fnos login setting changed", "module", "AUTH", "enabled", enabled)
	return nil
}

func (s *Service) FnosGatewayURL(ctx context.Context) (string, error) {
	return s.setting(ctx, settings.KeyFnosGatewayURL)
}

func (s *Service) SetFnosGatewayURL(ctx context.Context, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		_, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, settings.KeyFnosGatewayURL)
		return err
	}
	return s.setSetting(ctx, settings.KeyFnosGatewayURL, raw)
}

func (s *Service) HasFnosBinding(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM fnos_auth_bindings`).Scan(&n)
	return n > 0, err
}

func (s *Service) ListFnosBindings(ctx context.Context) ([]FnosBinding, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT nas_uid, nas_username, created_at
		FROM fnos_auth_bindings
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FnosBinding{}
	for rows.Next() {
		var item FnosBinding
		if err := rows.Scan(&item.UID, &item.Username, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// BindFnos attaches the gateway UID to the Fonu administrator.
// The UID comes from the trusted gateway, never from the request body.
func (s *Service) BindFnos(ctx context.Context, adminID int, uid, username string) error {
	uid = cleanUID(uid)
	if uid == "" {
		return errors.New("飞牛用户标识无效")
	}
	username = cleanName(username)
	var existing int
	err := s.db.QueryRowContext(ctx, `SELECT fonu_admin_id FROM fnos_auth_bindings WHERE nas_uid = ?`, uid).Scan(&existing)
	if err == nil {
		if existing != adminID {
			return errors.New("该飞牛账号已绑定其他管理员")
		}
		_, err = s.db.ExecContext(ctx, `UPDATE fnos_auth_bindings SET nas_username = ? WHERE nas_uid = ?`, username, uid)
		if err != nil {
			return err
		}
		slog.Info("fnos account binding refreshed", "module", "AUTH", "nas_uid", uid)
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO fnos_auth_bindings(nas_uid, nas_username, fonu_admin_id)
		VALUES (?, ?, ?)
	`, uid, username, adminID)
	if err != nil {
		return err
	}
	slog.Info("fnos account bound", "module", "AUTH", "nas_uid", uid, "nas_username", username)
	return nil
}

// UnbindFnos removes one NAS account, or every binding when uid is empty.
// Existing one-click sessions for that account are revoked. Password sessions stay.
func (s *Service) UnbindFnos(ctx context.Context, adminID int, uid string) error {
	uid = cleanUID(uid)
	if uid == "" {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM fnos_auth_bindings WHERE fonu_admin_id = ?`, adminID); err != nil {
			return err
		}
		if err := s.RevokeFnosSessions(ctx, ""); err != nil {
			return err
		}
		slog.Info("fnos account bindings cleared", "module", "AUTH")
		return nil
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM fnos_auth_bindings WHERE nas_uid = ? AND fonu_admin_id = ?`, uid, adminID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrFnosNotBound
	}
	if err := s.RevokeFnosSessions(ctx, uid); err != nil {
		return err
	}
	slog.Info("fnos account unbound", "module", "AUTH", "nas_uid", uid)
	return nil
}

// BoundAdmin returns the Fonu administrator for a gateway UID.
// A matching username is not enough; the UID must already be bound.
func (s *Service) BoundAdmin(ctx context.Context, uid, username string) (int, error) {
	uid = cleanUID(uid)
	if uid == "" {
		return 0, ErrFnosNotBound
	}
	var adminID int
	err := s.db.QueryRowContext(ctx, `SELECT fonu_admin_id FROM fnos_auth_bindings WHERE nas_uid = ?`, uid).Scan(&adminID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrFnosNotBound
	}
	if err != nil {
		return 0, err
	}
	username = cleanName(username)
	if username != "" {
		_, _ = s.db.ExecContext(ctx, `UPDATE fnos_auth_bindings SET nas_username = ? WHERE nas_uid = ?`, username, uid)
	}
	return adminID, nil
}

func (s *Service) RevokeFnosSessions(ctx context.Context, nasUID string) error {
	nasUID = cleanUID(nasUID)
	if nasUID == "" {
		_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE auth_source = ?`, SourceFnos)
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE auth_source = ? AND nas_uid = ?`, SourceFnos, nasUID)
	return err
}

func (s *Service) ConfirmPassword(ctx context.Context, adminID int, password string) error {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM admins WHERE id = ?`, adminID).Scan(&hash)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return errors.New("当前密码不正确")
	}
	return nil
}

func (s *Service) setting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func (s *Service) setSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

func cleanUID(uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" || len(uid) > 64 {
		return ""
	}
	for _, r := range uid {
		if r > unicode.MaxASCII || !(r == '.' || r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return ""
		}
	}
	return uid
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	runes := []rune(name)
	if len(runes) > 128 {
		name = string(runes[:128])
	}
	return name
}
