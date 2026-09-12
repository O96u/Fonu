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

func (s *Store) List(ctx context.Context) ([]Config, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, provider, root_domain, record_name, ipv4_enabled, ipv6_enabled, enabled,
		       CASE WHEN api_token_enc IS NOT NULL AND api_token_enc != '' THEN 1 ELSE 0 END,
		       COALESCE(last_ipv4, ''), COALESCE(last_ipv6, ''), last_status, COALESCE(last_error, ''), last_updated_at
		FROM ddns_configs ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []Config
	for rows.Next() {
		cfg, err := scanConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

func (s *Store) GetByID(ctx context.Context, id int64) (Config, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, provider, root_domain, record_name, ipv4_enabled, ipv6_enabled, enabled,
		       CASE WHEN api_token_enc IS NOT NULL AND api_token_enc != '' THEN 1 ELSE 0 END,
		       COALESCE(last_ipv4, ''), COALESCE(last_ipv6, ''), last_status, COALESCE(last_error, ''), last_updated_at
		FROM ddns_configs WHERE id = ?
	`, id)
	cfg, err := scanConfig(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Config{}, fmt.Errorf("DDNS 配置不存在")
	}
	return cfg, err
}

func (s *Store) GetByRootDomain(ctx context.Context, rootDomain string) (Config, error) {
	rootDomain = strings.ToLower(strings.TrimSpace(rootDomain))
	row := s.db.QueryRowContext(ctx, `
		SELECT id, provider, root_domain, record_name, ipv4_enabled, ipv6_enabled, enabled,
		       CASE WHEN api_token_enc IS NOT NULL AND api_token_enc != '' THEN 1 ELSE 0 END,
		       COALESCE(last_ipv4, ''), COALESCE(last_ipv6, ''), last_status, COALESCE(last_error, ''), last_updated_at
		FROM ddns_configs WHERE root_domain = ? ORDER BY id LIMIT 1
	`, rootDomain)
	cfg, err := scanConfig(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Config{}, fmt.Errorf("未找到域名 %s 的 DDNS 配置", rootDomain)
	}
	return cfg, err
}

func (s *Store) GetTokenByID(ctx context.Context, id int64) (string, error) {
	var enc string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(api_token_enc, '') FROM ddns_configs WHERE id = ?`, id).Scan(&enc)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("DDNS 配置不存在")
		}
		return "", err
	}
	return enc, nil
}

func (s *Store) Create(ctx context.Context, in SaveInput, tokenEnc string) (Config, error) {
	rootDomain, recordName, provider := normalizeInput(in)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO ddns_configs(provider, root_domain, record_name, ipv4_enabled, ipv6_enabled, enabled, api_token_enc, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, provider, rootDomain, recordName, boolInt(in.IPv4Enabled), boolInt(in.IPv6Enabled), boolInt(in.Enabled), tokenEnc)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Config{}, fmt.Errorf("该域名与子域名组合已存在")
		}
		return Config{}, err
	}
	id, _ := res.LastInsertId()
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, in SaveInput, tokenEnc string, updateToken bool) (Config, error) {
	rootDomain, recordName, provider := normalizeInput(in)
	if updateToken {
		_, err := s.db.ExecContext(ctx, `
			UPDATE ddns_configs
			SET provider = ?, root_domain = ?, record_name = ?, ipv4_enabled = ?, ipv6_enabled = ?, enabled = ?,
			    api_token_enc = ?, updated_at = datetime('now')
			WHERE id = ?
		`, provider, rootDomain, recordName, boolInt(in.IPv4Enabled), boolInt(in.IPv6Enabled), boolInt(in.Enabled), tokenEnc, id)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return Config{}, fmt.Errorf("该域名与子域名组合已存在")
			}
			return Config{}, err
		}
	} else {
		_, err := s.db.ExecContext(ctx, `
			UPDATE ddns_configs
			SET provider = ?, root_domain = ?, record_name = ?, ipv4_enabled = ?, ipv6_enabled = ?, enabled = ?,
			    updated_at = datetime('now')
			WHERE id = ?
		`, provider, rootDomain, recordName, boolInt(in.IPv4Enabled), boolInt(in.IPv6Enabled), boolInt(in.Enabled), id)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return Config{}, fmt.Errorf("该域名与子域名组合已存在")
			}
			return Config{}, err
		}
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM ddns_configs WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("DDNS 配置不存在")
	}
	return nil
}

func (s *Store) UpdateStatus(ctx context.Context, id int64, ipv4, ipv6, status, lastError string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE ddns_configs
		SET last_ipv4 = ?, last_ipv6 = ?, last_status = ?, last_error = ?, last_updated_at = datetime('now'), updated_at = datetime('now')
		WHERE id = ?
	`, ipv4, ipv6, status, lastError, id)
	return err
}

func normalizeInput(in SaveInput) (rootDomain, recordName, provider string) {
	rootDomain = strings.ToLower(strings.TrimSpace(in.RootDomain))
	recordName = strings.TrimSpace(in.RecordName)
	if recordName == "" {
		recordName = "*"
	}
	provider = strings.TrimSpace(in.Provider)
	if provider == "" {
		provider = "cloudflare"
	}
	return rootDomain, recordName, provider
}

func scanConfig(row interface{ Scan(dest ...any) error }) (Config, error) {
	var cfg Config
	var ipv4Enabled, ipv6Enabled, enabled, hasToken int
	var lastUpdated sql.NullString
	if err := row.Scan(&cfg.ID, &cfg.Provider, &cfg.RootDomain, &cfg.RecordName, &ipv4Enabled, &ipv6Enabled, &enabled, &hasToken, &cfg.LastIPv4, &cfg.LastIPv6, &cfg.LastStatus, &cfg.LastError, &lastUpdated); err != nil {
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
