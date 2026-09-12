package certificate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Record struct {
	ID          int64      `json:"id"`
	Domain      string     `json:"domain"`
	Domains     []string   `json:"domains,omitempty"`
	Wildcard    bool       `json:"wildcard"`
	ACMECA      string     `json:"acme_ca,omitempty"`
	CertPath    string     `json:"cert_path,omitempty"`
	KeyPath     string     `json:"key_path,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastRenewAt *time.Time `json:"last_renew_at,omitempty"`
	Status      string     `json:"status"`
	LastError   string     `json:"last_error,omitempty"`
	DaysLeft    int        `json:"days_left"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context) ([]Record, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, domain, COALESCE(domains, ''), wildcard, COALESCE(acme_ca, ''), COALESCE(cert_path, ''), COALESCE(key_path, ''),
		       expires_at, last_renew_at, status, COALESCE(last_error, '')
		FROM certificates ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (s *Store) Upsert(ctx context.Context, domain string, wildcard bool, domainsJSON, acmeCA, certPath, keyPath string, expiresAt time.Time, status string, lastError string) (Record, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO certificates(domain, domains, wildcard, acme_ca, cert_path, key_path, expires_at, last_renew_at, status, last_error, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?, datetime('now'))
		ON CONFLICT(domain) DO UPDATE SET
			domains = excluded.domains,
			wildcard = excluded.wildcard,
			acme_ca = excluded.acme_ca,
			cert_path = excluded.cert_path,
			key_path = excluded.key_path,
			expires_at = excluded.expires_at,
			last_renew_at = datetime('now'),
			status = excluded.status,
			last_error = excluded.last_error,
			updated_at = datetime('now')
	`, domain, domainsJSON, boolInt(wildcard), acmeCA, certPath, keyPath, expiresAt.UTC().Format(time.RFC3339), status, lastError)
	if err != nil {
		return Record{}, err
	}
	return s.GetByDomain(ctx, domain)
}

func (s *Store) UpdateStatus(ctx context.Context, domain, status, lastError string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE certificates SET status = ?, last_error = ?, updated_at = datetime('now') WHERE domain = ?
	`, status, lastError, domain)
	return err
}

func (s *Store) Delete(ctx context.Context, domain string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM certificates WHERE domain = ?`, domain)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("证书不存在")
	}
	return nil
}

func (s *Store) GetByDomain(ctx context.Context, domain string) (Record, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, domain, COALESCE(domains, ''), wildcard, COALESCE(acme_ca, ''), COALESCE(cert_path, ''), COALESCE(key_path, ''),
		       expires_at, last_renew_at, status, COALESCE(last_error, '')
		FROM certificates WHERE domain = ?
	`, domain)
	rec, err := scanRecord(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, fmt.Errorf("证书不存在")
	}
	return rec, err
}

func scanRecord(row interface{ Scan(dest ...any) error }) (Record, error) {
	var rec Record
	var wildcard int
	var domainsJSON string
	var expiresAt, lastRenewAt sql.NullString
	if err := row.Scan(&rec.ID, &rec.Domain, &domainsJSON, &wildcard, &rec.ACMECA, &rec.CertPath, &rec.KeyPath, &expiresAt, &lastRenewAt, &rec.Status, &rec.LastError); err != nil {
		return Record{}, err
	}
	rec.Wildcard = wildcard == 1
	rec.Domains = decodeDomains(domainsJSON, rec.Domain, rec.Wildcard)
	if expiresAt.Valid {
		t := parseTime(expiresAt.String)
		rec.ExpiresAt = &t
		rec.DaysLeft = int(time.Until(t).Hours() / 24)
	}
	if lastRenewAt.Valid {
		t := parseTime(lastRenewAt.String)
		rec.LastRenewAt = &t
	}
	return rec, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func decodeDomains(raw string, domain string, wildcard bool) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if wildcard {
			return []string{domain, "*." + domain}
		}
		return []string{domain}
	}
	var domains []string
	if err := json.Unmarshal([]byte(raw), &domains); err != nil || len(domains) == 0 {
		if wildcard {
			return []string{domain, "*." + domain}
		}
		return []string{domain}
	}
	return domains
}

func parseTime(v string) time.Time {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", v)
	}
	return t.UTC()
}
