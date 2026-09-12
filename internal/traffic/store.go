package traffic

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) LoadTotals(ctx context.Context) (map[string]hostTotals, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT host, upload_bytes, download_bytes FROM proxy_traffic`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]hostTotals{}
	for rows.Next() {
		var host string
		var up, down int64
		if err := rows.Scan(&host, &up, &down); err != nil {
			return nil, err
		}
		out[normalizeHost(host)] = hostTotals{Upload: up, Download: down}
	}
	return out, rows.Err()
}

func (s *Store) SaveTotals(ctx context.Context, hosts map[string]hostTotals) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for host, totals := range hosts {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO proxy_traffic(host, upload_bytes, download_bytes, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(host) DO UPDATE SET
				upload_bytes = excluded.upload_bytes,
				download_bytes = excluded.download_bytes,
				updated_at = excluded.updated_at
		`, host, totals.Upload, totals.Download, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type hostTotals struct {
	Upload   int64
	Download int64
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.Index(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}
