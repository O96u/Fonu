package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/fonu/fonu/internal/validate"
)

const sessionTTL = 7 * 24 * time.Hour

var ErrUnauthorized = errors.New("unauthorized")
var ErrNotInitialized = errors.New("not initialized")

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) IsInitialized(ctx context.Context) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM admins`).Scan(&count)
	return count > 0, err
}

const defaultAdminUsername = "admin"

func (s *Service) EnsureDefaultAdmin(ctx context.Context, log *slog.Logger) error {
	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		return err
	}
	if initialized {
		return nil
	}

	password, err := generatePassword(16)
	if err != nil {
		return err
	}
	if err := s.Setup(ctx, defaultAdminUsername, password); err != nil {
		return err
	}

	log.Info(
		"首次启动已创建管理员账户，请使用以下凭据登录，并在「设置」中尽快修改密码",
		"module", "SYSTEM",
		"username", defaultAdminUsername,
		"password", password,
	)
	return nil
}

func (s *Service) Setup(ctx context.Context, username, password string) error {
	if err := validate.Username(username); err != nil {
		return err
	}
	if err := validate.Password(password); err != nil {
		return err
	}

	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		return err
	}
	if initialized {
		return fmt.Errorf("管理员已初始化")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO admins(username, password_hash) VALUES (?, ?)`, username, string(hash))
	return err
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	var adminID int
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT id, password_hash FROM admins WHERE username = ?`, username).Scan(&adminID, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUnauthorized
		}
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", ErrUnauthorized
	}
	return s.createSession(ctx, adminID)
}

func (s *Service) AdminIDForSession(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, ErrUnauthorized
	}
	var adminID int
	err := s.db.QueryRowContext(ctx, `SELECT admin_id FROM sessions WHERE id = ?`, sessionID).Scan(&adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrUnauthorized
		}
		return 0, err
	}
	return adminID, nil
}

func (s *Service) ChangePassword(ctx context.Context, adminID int, oldPassword, newPassword string) error {
	if err := validate.Password(newPassword); err != nil {
		return err
	}
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM admins WHERE id = ?`, adminID).Scan(&hash)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("当前密码不正确")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE admins SET password_hash = ? WHERE id = ?`, string(newHash), adminID)
	return err
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

func (s *Service) ValidateSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return ErrUnauthorized
	}
	var expiresAt string
	err := s.db.QueryRowContext(ctx, `SELECT expires_at FROM sessions WHERE id = ?`, sessionID).Scan(&expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUnauthorized
		}
		return err
	}
	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return ErrUnauthorized
	}
	if time.Now().After(expires) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
		return ErrUnauthorized
	}
	return nil
}

func (s *Service) createSession(ctx context.Context, adminID int) (string, error) {
	id, err := randomToken(32)
	if err != nil {
		return "", err
	}
	expires := time.Now().Add(sessionTTL).UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx, `INSERT INTO sessions(id, admin_id, expires_at) VALUES (?, ?, ?)`, id, adminID, expires)
	if err != nil {
		return "", err
	}
	return id, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generatePassword(length int) (string, error) {
	const letters = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if length < 8 {
		length = 8
	}
	out := make([]byte, length)
	randBytes := make([]byte, length)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}
	for i := range out {
		out[i] = letters[int(randBytes[i])%len(letters)]
	}
	return string(out), nil
}
