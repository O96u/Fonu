package proxy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/validate"
)

type Rule struct {
	ID            int64     `json:"id"`
	Domain        string    `json:"domain"`
	Upstream      string    `json:"upstream"`
	HTTPSEnabled  bool      `json:"https_enabled"`
	HTTPRedirect  bool      `json:"http_redirect"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateInput struct {
	Domain       string
	Upstream     string
	HTTPSEnabled bool
	HTTPRedirect bool
	Enabled      bool
}

type UpdateInput struct {
	Domain       *string
	Upstream     *string
	HTTPSEnabled *bool
	HTTPRedirect *bool
	Enabled      *bool
}

type Store struct {
	db *sql.DB
	tx *sql.Tx
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func NewStoreWithTx(tx *sql.Tx) *Store {
	return &Store{tx: tx}
}

func (s *Store) querier() interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
} {
	if s.tx != nil {
		return s.tx
	}
	return s.db
}

func (s *Store) List(ctx context.Context) ([]Rule, error) {
	q := s.querier()
	rows, err := q.QueryContext(ctx, `
		SELECT id, domain, upstream, https_enabled, http_redirect, enabled, created_at, updated_at
		FROM proxy_rules
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) Get(ctx context.Context, id int64) (Rule, error) {
	row := s.querier().QueryRowContext(ctx, `
		SELECT id, domain, upstream, https_enabled, http_redirect, enabled, created_at, updated_at
		FROM proxy_rules WHERE id = ?
	`, id)
	rule, err := scanRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Rule{}, fmt.Errorf("规则不存在")
	}
	return rule, err
}

func (s *Store) Create(ctx context.Context, in CreateInput) (Rule, error) {
	domain := strings.ToLower(strings.TrimSpace(in.Domain))
	upstream, err := validate.Upstream(in.Upstream)
	if err != nil {
		return Rule{}, err
	}
	if err := validate.Domain(domain); err != nil {
		return Rule{}, err
	}

	res, err := s.querier().ExecContext(ctx, `
		INSERT INTO proxy_rules(domain, upstream, https_enabled, http_redirect, enabled, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
	`, domain, upstream, boolInt(in.HTTPSEnabled), boolInt(in.HTTPRedirect), boolInt(in.Enabled))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Rule{}, fmt.Errorf("域名 %s 已存在", domain)
		}
		return Rule{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Rule{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (Rule, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return Rule{}, err
	}

	domain := current.Domain
	upstream := current.Upstream
	httpsEnabled := current.HTTPSEnabled
	httpRedirect := current.HTTPRedirect
	enabled := current.Enabled

	if in.Domain != nil {
		domain = strings.ToLower(strings.TrimSpace(*in.Domain))
		if err := validate.Domain(domain); err != nil {
			return Rule{}, err
		}
	}
	if in.Upstream != nil {
		upstream, err = validate.Upstream(*in.Upstream)
		if err != nil {
			return Rule{}, err
		}
	}
	if in.HTTPSEnabled != nil {
		httpsEnabled = *in.HTTPSEnabled
	}
	if in.HTTPRedirect != nil {
		httpRedirect = *in.HTTPRedirect
	}
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	_, err = s.querier().ExecContext(ctx, `
		UPDATE proxy_rules
		SET domain = ?, upstream = ?, https_enabled = ?, http_redirect = ?, enabled = ?, updated_at = datetime('now')
		WHERE id = ?
	`, domain, upstream, boolInt(httpsEnabled), boolInt(httpRedirect), boolInt(enabled), id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Rule{}, fmt.Errorf("域名 %s 已存在", domain)
		}
		return Rule{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.querier().ExecContext(ctx, `DELETE FROM proxy_rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("规则不存在")
	}
	return nil
}

func (s *Store) ListEnabled(ctx context.Context) ([]Rule, error) {
	rows, err := s.querier().QueryContext(ctx, `
		SELECT id, domain, upstream, https_enabled, http_redirect, enabled, created_at, updated_at
		FROM proxy_rules WHERE enabled = 1 ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRule(row rowScanner) (Rule, error) {
	var rule Rule
	var httpsEnabled int
	var httpRedirect int
	var enabled int
	var createdAt string
	var updatedAt string
	if err := row.Scan(&rule.ID, &rule.Domain, &rule.Upstream, &httpsEnabled, &httpRedirect, &enabled, &createdAt, &updatedAt); err != nil {
		return Rule{}, err
	}
	rule.HTTPSEnabled = httpsEnabled == 1
	rule.HTTPRedirect = httpRedirect == 1
	rule.Enabled = enabled == 1
	rule.CreatedAt = parseTime(createdAt)
	rule.UpdatedAt = parseTime(updatedAt)
	return rule, nil
}

func parseTime(v string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", v)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, v)
	}
	return t.UTC()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
