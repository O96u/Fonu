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

type EntryCloneInput struct {
	Name         string
	ListenPort   int
	ListenIPv4   bool
	ListenIPv6   bool
	HTTPSEnabled bool
	HTTPRedirect bool
}

type cloneSourceRule struct {
	ID           int64
	Upstream     string
	Enabled      bool
	Name         string
	NginxMode    string
	SortOrder    int
	SecurityJSON string
	Hostnames    []string
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
	if err := s.ensureEntryListenPortAvailable(ctx, in.ListenPort, 0); err != nil {
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

	if listenPort != current.ListenPort {
		if err := s.ensureEntryListenPortAvailable(ctx, listenPort, id); err != nil {
			return Entry{}, err
		}
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

// CloneEntry duplicates an entry together with all its rules (frontend
// hosts and security settings included). The clone listens on the settings
// provided in in. It returns the new entry and a mapping from source rule
// IDs to cloned rule IDs.
func (s *Store) CloneEntry(ctx context.Context, sourceID int64, in EntryCloneInput) (Entry, map[int64]int64, error) {
	if _, err := s.GetEntry(ctx, sourceID); err != nil {
		return Entry{}, nil, err
	}
	if err := validateEntryInput(in.ListenPort, in.ListenIPv4, in.ListenIPv6); err != nil {
		return Entry{}, nil, err
	}
	if err := s.ensureEntryListenPortAvailable(ctx, in.ListenPort, sourceID); err != nil {
		return Entry{}, nil, err
	}
	name, err := normalizeName(in.Name)
	if err != nil {
		return Entry{}, nil, err
	}

	sourceRules, err := s.loadCloneSourceRules(ctx, sourceID)
	if err != nil {
		return Entry{}, nil, err
	}
	if conflict, err := s.clonePortConflicts(ctx, in.ListenPort, sourceRules); err != nil {
		return Entry{}, nil, err
	} else if conflict != "" {
		return Entry{}, nil, fmt.Errorf("%s", conflict)
	}

	sortOrder, err := s.nextEntrySortOrder(ctx)
	if err != nil {
		return Entry{}, nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO proxy_entries(name, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, sort_order, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, name, in.ListenPort, boolInt(in.ListenIPv4), boolInt(in.ListenIPv6), boolInt(in.HTTPSEnabled), boolInt(in.HTTPRedirect), sortOrder)
	if err != nil {
		return Entry{}, nil, err
	}
	newEntryID, err := res.LastInsertId()
	if err != nil {
		return Entry{}, nil, err
	}

	ruleIDMap := make(map[int64]int64, len(sourceRules))
	for _, src := range sourceRules {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO proxy_rules(entry_id, upstream, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, enabled, nginx_mode, name, sort_order, security_json, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		`, newEntryID, src.Upstream, in.ListenPort, boolInt(in.ListenIPv4), boolInt(in.ListenIPv6), boolInt(in.HTTPSEnabled), boolInt(in.HTTPRedirect), boolInt(src.Enabled), src.NginxMode, src.Name, src.SortOrder, src.SecurityJSON)
		if err != nil {
			return Entry{}, nil, err
		}
		newRuleID, err := res.LastInsertId()
		if err != nil {
			return Entry{}, nil, err
		}
		ruleIDMap[src.ID] = newRuleID
		for _, hostname := range src.Hostnames {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO proxy_hosts(rule_id, hostname, listen_port)
				VALUES (?, ?, ?)
			`, newRuleID, hostname, in.ListenPort); err != nil {
				return Entry{}, nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return Entry{}, nil, err
	}
	entry, err := s.GetEntry(ctx, newEntryID)
	if err != nil {
		return Entry{}, nil, err
	}
	return entry, ruleIDMap, nil
}

func (s *Store) loadCloneSourceRules(ctx context.Context, entryID int64) ([]cloneSourceRule, error) {
	rows, err := s.querier().QueryContext(ctx, `
		SELECT id, upstream, enabled, name, nginx_mode, sort_order, security_json
		FROM proxy_rules
		WHERE entry_id = ?
		ORDER BY sort_order ASC, id ASC
	`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []cloneSourceRule
	for rows.Next() {
		var rule cloneSourceRule
		var enabled int
		if err := rows.Scan(&rule.ID, &rule.Upstream, &enabled, &rule.Name, &rule.NginxMode, &rule.SortOrder, &rule.SecurityJSON); err != nil {
			return nil, err
		}
		rule.Enabled = enabled == 1
		if rule.NginxMode == "" {
			rule.NginxMode = "auto"
		}
		out = append(out, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		hostRows, err := s.querier().QueryContext(ctx, `
			SELECT hostname FROM proxy_hosts WHERE rule_id = ? ORDER BY id ASC
		`, out[i].ID)
		if err != nil {
			return nil, err
		}
		var names []string
		for hostRows.Next() {
			var hostname string
			if err := hostRows.Scan(&hostname); err != nil {
				hostRows.Close()
				return nil, err
			}
			names = append(names, hostname)
		}
		hostRows.Close()
		if err := hostRows.Err(); err != nil {
			return nil, err
		}
		out[i].Hostnames = names
	}
	return out, nil
}

func (s *Store) clonePortConflicts(ctx context.Context, port int, rules []cloneSourceRule) (string, error) {
	seen := make(map[string]struct{})
	names := make([]string, 0)
	for _, rule := range rules {
		for _, hostname := range rule.Hostnames {
			key := strings.ToLower(strings.TrimSpace(hostname))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			names = append(names, hostname)
		}
	}
	if len(names) == 0 {
		return "", nil
	}

	placeholders := make([]string, len(names))
	args := make([]any, 0, len(names)+1)
	args = append(args, port)
	for i, name := range names {
		placeholders[i] = "?"
		args = append(args, name)
	}

	rows, err := s.querier().QueryContext(ctx, `
		SELECT DISTINCT hostname
		FROM proxy_hosts
		WHERE listen_port = ? AND hostname COLLATE NOCASE IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var conflicts []string
	for rows.Next() {
		var hostname string
		if err := rows.Scan(&hostname); err != nil {
			return "", err
		}
		conflicts = append(conflicts, hostname)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(conflicts) == 0 {
		return "", nil
	}
	if len(conflicts) > 5 {
		conflicts = append(conflicts[:5], "…")
	}
	return fmt.Sprintf("目标端口 %d 上以下域名已被其他规则使用：%s，请更换端口或调整原规则", port, strings.Join(conflicts, "、")), nil
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

func entryDisplayLabel(entry Entry) string {
	name := strings.TrimSpace(entry.Name)
	if name != "" {
		return name
	}
	return fmt.Sprintf("端口 %d", entry.ListenPort)
}

func (s *Store) ensureEntryListenPortAvailable(ctx context.Context, listenPort int, excludeID int64) error {
	entries, err := s.ListEntries(ctx)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if excludeID > 0 && entry.ID == excludeID {
			continue
		}
		if entry.ListenPort == listenPort {
			return fmt.Errorf("监听端口 %d 已被入口「%s」占用，请更换端口", listenPort, entryDisplayLabel(entry))
		}
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
