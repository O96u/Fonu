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

type Entry struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ListenPort   int       `json:"listen_port"`
	ListenIPv4   bool      `json:"listen_ipv4"`
	ListenIPv6   bool      `json:"listen_ipv6"`
	HTTPSEnabled bool      `json:"https_enabled"`
	HTTPRedirect bool      `json:"http_redirect"`
	SortOrder    int       `json:"sort_order"`
	RuleCount    int       `json:"rule_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EntryCreateInput struct {
	Name         string
	ListenPort   int
	ListenIPv4   bool
	ListenIPv6   bool
	HTTPSEnabled bool
	HTTPRedirect bool
}

type EntryUpdateInput struct {
	Name         *string
	ListenPort   *int
	ListenIPv4   *bool
	ListenIPv6   *bool
	HTTPSEnabled *bool
	HTTPRedirect *bool
}

func (s *Store) ListEntries(ctx context.Context) ([]Entry, error) {
	rows, err := s.querier().QueryContext(ctx, `
		SELECT
			e.id, e.name, e.listen_port, e.listen_ipv4, e.listen_ipv6,
			e.https_enabled, e.http_redirect, e.sort_order, e.created_at, e.updated_at,
			(SELECT COUNT(*) FROM proxy_rules r WHERE r.entry_id = e.id) AS rule_count
		FROM proxy_entries e
		ORDER BY e.sort_order ASC, e.listen_port ASC, e.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) GetEntry(ctx context.Context, id int64) (Entry, error) {
	row := s.querier().QueryRowContext(ctx, `
		SELECT
			e.id, e.name, e.listen_port, e.listen_ipv4, e.listen_ipv6,
			e.https_enabled, e.http_redirect, e.sort_order, e.created_at, e.updated_at,
			(SELECT COUNT(*) FROM proxy_rules r WHERE r.entry_id = e.id) AS rule_count
		FROM proxy_entries e
		WHERE e.id = ?
	`, id)
	entry, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, fmt.Errorf("入口不存在")
	}
	return entry, err
}

func (s *Store) CreateEntry(ctx context.Context, in EntryCreateInput) (Entry, error) {
	if err := validateEntryInput(in.ListenPort, in.ListenIPv4, in.ListenIPv6); err != nil {
		return Entry{}, err
	}
	name, err := normalizeName(in.Name)
	if err != nil {
		return Entry{}, err
	}
	sortOrder, err := s.nextEntrySortOrder(ctx)
	if err != nil {
		return Entry{}, err
	}

	res, err := s.querier().ExecContext(ctx, `
		INSERT INTO proxy_entries(name, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, sort_order, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, name, in.ListenPort, boolInt(in.ListenIPv4), boolInt(in.ListenIPv6), boolInt(in.HTTPSEnabled), boolInt(in.HTTPRedirect), sortOrder)
	if err != nil {
		return Entry{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Entry{}, err
	}
	return s.GetEntry(ctx, id)
}

func (s *Store) UpdateEntry(ctx context.Context, id int64, in EntryUpdateInput) (Entry, error) {
	current, err := s.GetEntry(ctx, id)
	if err != nil {
		return Entry{}, err
	}

	name := current.Name
	listenPort := current.ListenPort
	listenIPv4 := current.ListenIPv4
	listenIPv6 := current.ListenIPv6
	httpsEnabled := current.HTTPSEnabled
	httpRedirect := current.HTTPRedirect

	if in.Name != nil {
		name, err = normalizeName(*in.Name)
		if err != nil {
			return Entry{}, err
		}
	}
	if in.ListenPort != nil {
		if err := validate.ListenPort(*in.ListenPort); err != nil {
			return Entry{}, err
		}
		listenPort = *in.ListenPort
	}
	if in.ListenIPv4 != nil {
		listenIPv4 = *in.ListenIPv4
	}
	if in.ListenIPv6 != nil {
		listenIPv6 = *in.ListenIPv6
	}
	if err := validateEntryInput(listenPort, listenIPv4, listenIPv6); err != nil {
		return Entry{}, err
	}
	if in.HTTPSEnabled != nil {
		httpsEnabled = *in.HTTPSEnabled
	}
	if in.HTTPRedirect != nil {
		httpRedirect = *in.HTTPRedirect
	}

	_, err = s.querier().ExecContext(ctx, `
		UPDATE proxy_entries
		SET name = ?, listen_port = ?, listen_ipv4 = ?, listen_ipv6 = ?,
		    https_enabled = ?, http_redirect = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, listenPort, boolInt(listenIPv4), boolInt(listenIPv6), boolInt(httpsEnabled), boolInt(httpRedirect), id)
	if err != nil {
		return Entry{}, err
	}

	if err := s.syncRulesFromEntry(ctx, id, listenPort, listenIPv4, listenIPv6, httpsEnabled, httpRedirect); err != nil {
		return Entry{}, err
	}
	return s.GetEntry(ctx, id)
}

func (s *Store) DeleteEntry(ctx context.Context, id int64) error {
	if _, err := s.GetEntry(ctx, id); err != nil {
		return err
	}
	if _, err := s.querier().ExecContext(ctx, `DELETE FROM proxy_rules WHERE entry_id = ?`, id); err != nil {
		return err
	}
	res, err := s.querier().ExecContext(ctx, `DELETE FROM proxy_entries WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("入口不存在")
	}
	return nil
}

func (s *Store) RuleIDsByEntry(ctx context.Context, entryID int64) ([]int64, error) {
	rows, err := s.querier().QueryContext(ctx, `SELECT id FROM proxy_rules WHERE entry_id = ? ORDER BY sort_order ASC, id ASC`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) findOrCreateEntryForListen(ctx context.Context, listenPort int, listenIPv4, listenIPv6, httpsEnabled, httpRedirect bool) (*int64, error) {
	var id int64
	err := s.querier().QueryRowContext(ctx, `
		SELECT id FROM proxy_entries
		WHERE listen_port = ? AND listen_ipv4 = ? AND listen_ipv6 = ?
		  AND https_enabled = ? AND http_redirect = ?
		LIMIT 1
	`, listenPort, boolInt(listenIPv4), boolInt(listenIPv6), boolInt(httpsEnabled), boolInt(httpRedirect)).Scan(&id)
	if err == nil {
		return &id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	entry, err := s.CreateEntry(ctx, EntryCreateInput{
		ListenPort:   listenPort,
		ListenIPv4:   listenIPv4,
		ListenIPv6:   listenIPv6,
		HTTPSEnabled: httpsEnabled,
		HTTPRedirect: httpRedirect,
	})
	if err != nil {
		return nil, err
	}
	return &entry.ID, nil
}

func (s *Store) syncRulesFromEntry(ctx context.Context, entryID int64, listenPort int, listenIPv4, listenIPv6, httpsEnabled, httpRedirect bool) error {
	_, err := s.querier().ExecContext(ctx, `
		UPDATE proxy_rules
		SET listen_port = ?, listen_ipv4 = ?, listen_ipv6 = ?,
		    https_enabled = ?, http_redirect = ?, updated_at = datetime('now')
		WHERE entry_id = ?
	`, listenPort, boolInt(listenIPv4), boolInt(listenIPv6), boolInt(httpsEnabled), boolInt(httpRedirect), entryID)
	return err
}

func validateEntryInput(listenPort int, listenIPv4, listenIPv6 bool) error {
	if err := validate.ListenPort(listenPort); err != nil {
		return err
	}
	if !listenIPv4 && !listenIPv6 {
		return fmt.Errorf("至少需要启用 IPv4 或 IPv6 监听")
	}
	return nil
}

func (s *Store) nextEntrySortOrder(ctx context.Context) (int, error) {
	var maxOrder sql.NullInt64
	err := s.querier().QueryRowContext(ctx, `SELECT MAX(sort_order) FROM proxy_entries`).Scan(&maxOrder)
	if err != nil {
		return 0, err
	}
	if !maxOrder.Valid {
		return 0, nil
	}
	return int(maxOrder.Int64) + 1, nil
}

func (s *Store) ReorderEntries(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("无效的入口 ID")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("排序列表包含重复的入口 ID")
		}
		seen[id] = struct{}{}
	}

	var total int
	if err := s.querier().QueryRowContext(ctx, `SELECT COUNT(*) FROM proxy_entries`).Scan(&total); err != nil {
		return err
	}
	if len(ids) != total {
		return fmt.Errorf("排序列表必须包含全部入口")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for index, id := range ids {
		res, err := tx.ExecContext(ctx, `
			UPDATE proxy_entries SET sort_order = ?, updated_at = datetime('now') WHERE id = ?
		`, index, id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("入口不存在")
		}
	}
	return tx.Commit()
}

func scanEntry(row rowScanner) (Entry, error) {
	var entry Entry
	var listenIPv4 int
	var listenIPv6 int
	var httpsEnabled int
	var httpRedirect int
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&entry.ID,
		&entry.Name,
		&entry.ListenPort,
		&listenIPv4,
		&listenIPv6,
		&httpsEnabled,
		&httpRedirect,
		&entry.SortOrder,
		&createdAt,
		&updatedAt,
		&entry.RuleCount,
	); err != nil {
		return Entry{}, err
	}
	entry.ListenIPv4 = listenIPv4 == 1
	entry.ListenIPv6 = listenIPv6 == 1
	entry.HTTPSEnabled = httpsEnabled == 1
	entry.HTTPRedirect = httpRedirect == 1
	entry.Name = strings.TrimSpace(entry.Name)
	entry.CreatedAt = parseTime(createdAt)
	entry.UpdatedAt = parseTime(updatedAt)
	return entry, nil
}

func EntriesForAPI(entries []Entry) []Entry {
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out
}
