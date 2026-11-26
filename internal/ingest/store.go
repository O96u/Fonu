package ingest

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/logstore"
)

type FactRow struct {
	TS            time.Time
	ClientIP      string
	Host          string
	Service       string
	Method        string
	Path          string
	Status        int
	ServerPort    int
	RequestLength int64
	BytesSent     int64
	RequestMS     float64
	UserAgent     string
	Kind          string
	GeoCountry    string
	GeoProvince   string
	GeoCity       string
	GeoLabel      string
	GeoPending    bool
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type IngestState struct {
	Offset    int64
	FileSize  int64
	MtimeNs   int64
	UpdatedAt time.Time
}

func (s *Store) MaxTS(ctx context.Context) (time.Time, bool) {
	var raw sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT MAX(ts) FROM access_facts`).Scan(&raw)
	if err != nil || !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw.String)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func (s *Store) LoadState(ctx context.Context, logPath string) (IngestState, error) {
	var offset, size, mtime int64
	var updated string
	err := s.db.QueryRowContext(ctx, `
		SELECT file_offset, file_size, file_mtime_ns, updated_at
		FROM ingest_state WHERE log_path = ?
	`, logPath).Scan(&offset, &size, &mtime, &updated)
	if err == sql.ErrNoRows {
		return IngestState{}, nil
	}
	if err != nil {
		return IngestState{}, err
	}
	t, _ := time.Parse(time.RFC3339, updated)
	return IngestState{Offset: offset, FileSize: size, MtimeNs: mtime, UpdatedAt: t}, nil
}

func (s *Store) SaveState(ctx context.Context, logPath string, st IngestState) error {
	return saveState(ctx,s.db,logPath,st)
}

type stateExecutor interface {
	ExecContext(context.Context,string,...any)(sql.Result,error)
}

func saveState(ctx context.Context, executor stateExecutor, logPath string, st IngestState) error {
	_, err := executor.ExecContext(ctx, `
		INSERT INTO ingest_state(log_path, file_offset, file_size, file_mtime_ns, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(log_path) DO UPDATE SET
			file_offset = excluded.file_offset,
			file_size = excluded.file_size,
			file_mtime_ns = excluded.file_mtime_ns,
			updated_at = excluded.updated_at
	`, logPath, st.Offset, st.FileSize, st.MtimeNs, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) InsertBatch(ctx context.Context, rows []FactRow) error {
	return s.insertBatch(ctx,rows,"",nil)
}

// InsertBatchAndState commits the facts and their consumed log offset together.
func (s *Store) InsertBatchAndState(ctx context.Context, rows []FactRow, path string, st IngestState) error {
	return s.insertBatch(ctx,rows,path,&st)
}

func (s *Store) insertBatch(ctx context.Context, rows []FactRow, path string, st *IngestState) error {
	if len(rows)==0 && st==nil {return nil}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO access_facts(
			ts, client_ip, host, service, method, path, status, bytes_sent, request_ms,
			user_agent, kind, geo_country, geo_province, geo_city, geo_label, geo_pending
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, row := range rows {
		pending := 0
		if row.GeoPending {
			pending = 1
		}
		if _, err := stmt.ExecContext(ctx,
			row.TS.UTC().Format(time.RFC3339),
			row.ClientIP,
			row.Host,
			row.Service,
			row.Method,
			row.Path,
			row.Status,
			row.BytesSent,
			row.RequestMS,
			row.UserAgent,
			row.Kind,
			row.GeoCountry,
			row.GeoProvince,
			row.GeoCity,
			row.GeoLabel,
			pending,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if st!=nil {if err:=saveState(ctx,tx,path,*st);err!=nil{return err}}
	return tx.Commit()
}

func (s *Store) LoadGeoCache(ctx context.Context, ip string) (FactRow, bool, error) {
	var country, province, city, label string
	err := s.db.QueryRowContext(ctx, `
		SELECT geo_country, geo_province, geo_city, geo_label FROM geo_cache WHERE ip = ?
	`, ip).Scan(&country, &province, &city, &label)
	if err == sql.ErrNoRows {
		return FactRow{}, false, nil
	}
	if err != nil {
		return FactRow{}, false, err
	}
	return FactRow{GeoCountry: country, GeoProvince: province, GeoCity: city, GeoLabel: label}, true, nil
}

func (s *Store) SaveGeoCache(ctx context.Context, ip string, row FactRow) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO geo_cache(ip, geo_country, geo_province, geo_city, geo_label, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(ip) DO UPDATE SET
			geo_country = excluded.geo_country,
			geo_province = excluded.geo_province,
			geo_city = excluded.geo_city,
			geo_label = excluded.geo_label,
			updated_at = excluded.updated_at
	`, ip, row.GeoCountry, row.GeoProvince, row.GeoCity, row.GeoLabel, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) PendingIPs(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT client_ip FROM access_facts
		WHERE geo_pending = 1 AND client_ip != ''
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

func (s *Store) ApplyGeo(ctx context.Context, ip string, row FactRow) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE access_facts SET
			geo_country = ?,
			geo_province = ?,
			geo_city = ?,
			geo_label = ?,
			geo_pending = 0
		WHERE client_ip = ? AND geo_pending = 1
	`, row.GeoCountry, row.GeoProvince, row.GeoCity, row.GeoLabel, ip)
	return err
}

func (s *Store) PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	const chunk=2000
 var total int64
 for {
  if err:=ctx.Err();err!=nil{return total,err}
  res,err:=s.db.ExecContext(ctx,"DELETE FROM access_facts WHERE id IN (SELECT id FROM access_facts WHERE ts < ? ORDER BY ts LIMIT ?)",cutoff.UTC().Format(time.RFC3339),chunk)
  if err!=nil{return total,err}
  n,err:=res.RowsAffected();if err!=nil{return total,err}
  total+=n
  if n<chunk{return total,nil}
  select{case <-ctx.Done():return total,ctx.Err();case <-time.After(10*time.Millisecond):}
 }
}

// ParsedLine is exported for traffic subscribers.
type ParsedLine struct {
	Entry logstore.AccessEntry
	At    time.Time
}
