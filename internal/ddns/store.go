package ddns

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Config struct {
	ID            int64      `json:"id"`
	Provider      string     `json:"provider"`
	RootDomain    string     `json:"root_domain"`
	RecordName    string     `json:"record_name"`
	IPv4Enabled   bool       `json:"ipv4_enabled"`
	IPv6Enabled   bool       `json:"ipv6_enabled"`
	Enabled       bool       `json:"enabled"`
	HasToken      bool       `json:"has_token"`
	LastIPv4      string     `json:"last_ipv4"`
	LastIPv6      string     `json:"last_ipv6"`
	LastStatus    string     `json:"last_status"`
	LastError     string     `json:"last_error,omitempty"`
	LastUpdatedAt *time.Time `json:"last_updated_at,omitempty"`
}

type SaveInput struct {
	Provider    string
	RootDomain  string
	RecordName  string
	IPv4Enabled bool
	IPv6Enabled bool
	Enabled     bool
	APIToken    string
	APITokenID  string
	APISecret   string
}

func (in SaveInput) HasCredentialUpdate() bool {
	return strings.TrimSpace(in.APIToken) != "" ||
		strings.TrimSpace(in.APITokenID) != "" ||
		strings.TrimSpace(in.APISecret) != ""
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Get(ctx context.Context) (Config, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, provider, root_domain, record_name, ipv4_enabled, ipv6_enabled, enabled,
		       CASE WHEN api_token_enc IS NOT NULL AND api_token_enc != '' THEN 1 ELSE 0 END,
		       COALESCE(last_ipv4, ''), COALESCE(last_ipv6, ''), last_status, COALESCE(last_error, ''), last_updated_at
		FROM ddns_configs ORDER BY id LIMIT 1
	`)
	return scanConfig(row)
}

func (s *Store) GetToken(ctx context.Context) (string, error) {
	var enc string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(api_token_enc, '') FROM ddns_configs ORDER BY id LIMIT 1`).Scan(&enc)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return enc, nil
}

func (s *Store) Save(ctx context.Context, in SaveInput, tokenEnc string) (Config, error) {
	rootDomain := strings.ToLower(strings.TrimSpace(in.RootDomain))
	recordName := strings.TrimSpace(in.RecordName)
	if recordName == "" {
		recordName = "*"
	}
	provider := strings.TrimSpace(in.Provider)
	if provider == "" {
		provider = "cloudflare"
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM ddns_configs`).Scan(&count); err != nil {
		return Config{}, err
	}

	if count == 0 {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO ddns_configs(provider, root_domain, record_name, ipv4_enabled, ipv6_enabled, enabled, api_token_enc, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		`, provider, rootDomain, recordName, boolInt(in.IPv4Enabled), boolInt(in.IPv6Enabled), boolInt(in.Enabled), tokenEnc)
		if err != nil {
			return Config{}, err
		}
		return s.Get(ctx)
	}

	if in.HasCredentialUpdate() {
		_, err := s.db.ExecContext(ctx, `
			UPDATE ddns_configs
			SET provider = ?, root_domain = ?, record_name = ?, ipv4_enabled = ?, ipv6_enabled = ?, enabled = ?,
			    api_token_enc = ?, updated_at = datetime('now')
		`, provider, rootDomain, recordName, boolInt(in.IPv4Enabled), boolInt(in.IPv6Enabled), boolInt(in.Enabled), tokenEnc)
		if err != nil {
			return Config{}, err
		}
	} else {
		_, err := s.db.ExecContext(ctx, `
			UPDATE ddns_configs
			SET provider = ?, root_domain = ?, record_name = ?, ipv4_enabled = ?, ipv6_enabled = ?, enabled = ?,
			    updated_at = datetime('now')
		`, provider, rootDomain, recordName, boolInt(in.IPv4Enabled), boolInt(in.IPv6Enabled), boolInt(in.Enabled))
		if err != nil {
			return Config{}, err
		}
	}
	return s.Get(ctx)
}

func (s *Store) UpdateStatus(ctx context.Context, ipv4, ipv6, status, lastError string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE ddns_configs
		SET last_ipv4 = ?, last_ipv6 = ?, last_status = ?, last_error = ?, last_updated_at = datetime('now'), updated_at = datetime('now')
		WHERE id = (SELECT id FROM ddns_configs ORDER BY id LIMIT 1)
	`, ipv4, ipv6, status, lastError)
	return err
}

func scanConfig(row interface{ Scan(dest ...any) error }) (Config, error) {
	var cfg Config
	var ipv4Enabled, ipv6Enabled, enabled, hasToken int
	var lastUpdated sql.NullString
	if err := row.Scan(&cfg.ID, &cfg.Provider, &cfg.RootDomain, &cfg.RecordName, &ipv4Enabled, &ipv6Enabled, &enabled, &hasToken, &cfg.LastIPv4, &cfg.LastIPv6, &cfg.LastStatus, &cfg.LastError, &lastUpdated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Config{}, fmt.Errorf("DDNS 未配置")
		}
		return Config{}, err
	}
	cfg.IPv4Enabled = ipv4Enabled == 1
	cfg.IPv6Enabled = ipv6Enabled == 1
	cfg.Enabled = enabled == 1
	cfg.HasToken = hasToken == 1
	if lastUpdated.Valid {
		t := parseTime(lastUpdated.String)
		cfg.LastUpdatedAt = &t
	}
	return cfg, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func parseTime(v string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", v)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, v)
	}
	return t.UTC()
}
