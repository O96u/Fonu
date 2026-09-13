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

type Host struct {
	ID         int64  `json:"id"`
	Hostname   string `json:"hostname"`
	ListenPort *int   `json:"listen_port,omitempty"`
}

type Rule struct {
	ID           int64     `json:"id"`
	Domain       string    `json:"domain"`
	Upstream     string    `json:"upstream"`
	ListenPort   int       `json:"listen_port"`
	ListenIPv4   bool      `json:"listen_ipv4"`
	ListenIPv6   bool      `json:"listen_ipv6"`
	Hosts        []Host    `json:"hosts"`
	HTTPSEnabled bool      `json:"https_enabled"`
	HTTPRedirect bool      `json:"http_redirect"`
	Enabled      bool      `json:"enabled"`
	Remark       string    `json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PortGroup struct {
	Port      int
	Hostnames []string
}

func (r Rule) PrimaryHost() string {
	if len(r.Hosts) > 0 {
		return r.Hosts[0].Hostname
	}
	return ""
}

func (r Rule) Hostnames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, host := range r.Hosts {
		name := strings.ToLower(strings.TrimSpace(host.Hostname))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	if len(names) == 0 {
		domain := strings.ToLower(strings.TrimSpace(r.Domain))
		if domain != "" {
			names = append(names, domain)
		}
	}
	return names
}

func (r Rule) PortGroups() []PortGroup {
	groups := map[int][]string{}
	ports := make([]int, 0)
	for _, host := range r.Hosts {
		port := r.ListenPort
		if host.ListenPort != nil && *host.ListenPort > 0 {
			port = *host.ListenPort
		}
		if _, ok := groups[port]; !ok {
			ports = append(ports, port)
		}
		groups[port] = append(groups[port], host.Hostname)
	}

	out := make([]PortGroup, 0, len(ports))
	for _, port := range ports {
		out = append(out, PortGroup{Port: port, Hostnames: groups[port]})
	}
	return out
}

type CreateInput struct {
	Upstream     string
	ListenPort   int
	ListenIPv4   bool
	ListenIPv6   bool
	Hosts        []string
	HTTPSEnabled bool
	HTTPRedirect bool
	Enabled      bool
	Remark       string
}

type UpdateInput struct {
	Upstream     *string
	ListenPort   *int
	ListenIPv4   *bool
	ListenIPv6   *bool
	Hosts        *[]string
	HTTPSEnabled *bool
	HTTPRedirect *bool
	Enabled      *bool
	Remark       *string
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
	if err := s.cleanupOrphanRules(ctx); err != nil {
		return nil, err
	}
	q := s.querier()
	rows, err := q.QueryContext(ctx, `
		SELECT id, upstream, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, enabled, remark, created_at, updated_at
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.attachHosts(ctx, rules)
}

func (s *Store) Get(ctx context.Context, id int64) (Rule, error) {
	row := s.querier().QueryRowContext(ctx, `
		SELECT id, upstream, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, enabled, remark, created_at, updated_at
		FROM proxy_rules WHERE id = ?
	`, id)
	rule, err := scanRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Rule{}, fmt.Errorf("规则不存在")
	}
	if err != nil {
		return Rule{}, err
	}
	rules, err := s.attachHosts(ctx, []Rule{rule})
	if err != nil {
		return Rule{}, err
	}
	return rules[0], nil
}

func (s *Store) Create(ctx context.Context, in CreateInput) (Rule, error) {
	upstream, hosts, err := validateCreateInput(in)
	if err != nil {
		return Rule{}, err
	}
	if err := s.ensureHostsAvailable(ctx, hosts, in.ListenPort, 0); err != nil {
		return Rule{}, err
	}

	remark, err := normalizeRemark(in.Remark)
	if err != nil {
		return Rule{}, err
	}

	res, err := s.querier().ExecContext(ctx, `
		INSERT INTO proxy_rules(upstream, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, enabled, remark, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, upstream, in.ListenPort, boolInt(in.ListenIPv4), boolInt(in.ListenIPv6), boolInt(in.HTTPSEnabled), boolInt(in.HTTPRedirect), boolInt(in.Enabled), remark)
	if err != nil {
		return Rule{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Rule{}, err
	}
	if err := s.replaceHosts(ctx, id, in.ListenPort, hosts); err != nil {
		_, _ = s.querier().ExecContext(ctx, `DELETE FROM proxy_rules WHERE id = ?`, id)
		return Rule{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (Rule, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return Rule{}, err
	}

	oldListenPort := current.ListenPort
	upstream := current.Upstream
	listenPort := current.ListenPort
	listenIPv4 := current.ListenIPv4
	listenIPv6 := current.ListenIPv6
	httpsEnabled := current.HTTPSEnabled
	httpRedirect := current.HTTPRedirect
	enabled := current.Enabled
	remark := current.Remark
	hosts := current.Hosts

	if in.Upstream != nil {
		upstream, err = validate.Upstream(*in.Upstream)
		if err != nil {
			return Rule{}, err
		}
	}
	if in.ListenPort != nil {
		if err := validate.ListenPort(*in.ListenPort); err != nil {
			return Rule{}, err
		}
		listenPort = *in.ListenPort
	}
	if in.ListenIPv4 != nil {
		listenIPv4 = *in.ListenIPv4
	}
	if in.ListenIPv6 != nil {
		listenIPv6 = *in.ListenIPv6
	}
	if !listenIPv4 && !listenIPv6 {
		return Rule{}, fmt.Errorf("至少需要启用 IPv4 或 IPv6 监听")
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
	if in.Remark != nil {
		remark, err = normalizeRemark(*in.Remark)
		if err != nil {
			return Rule{}, err
		}
	}
	if in.Hosts != nil {
		hosts, err = parseHosts(*in.Hosts, listenPort)
		if err != nil {
			return Rule{}, err
		}
		if len(hosts) == 0 {
			return Rule{}, fmt.Errorf("至少需要一个前端域名")
		}
	}

	if in.ListenPort != nil && *in.ListenPort != oldListenPort && in.Hosts == nil {
		for _, host := range current.Hosts {
			if hostEffectivePort(host, oldListenPort) != oldListenPort {
				continue
			}
			if err := s.ensureHostAvailable(ctx, host.Hostname, listenPort, id); err != nil {
				return Rule{}, err
			}
		}
	}

	_, err = s.querier().ExecContext(ctx, `
		UPDATE proxy_rules
		SET upstream = ?, listen_port = ?, listen_ipv4 = ?, listen_ipv6 = ?, https_enabled = ?, http_redirect = ?, enabled = ?, remark = ?, updated_at = datetime('now')
		WHERE id = ?
	`, upstream, listenPort, boolInt(listenIPv4), boolInt(listenIPv6), boolInt(httpsEnabled), boolInt(httpRedirect), boolInt(enabled), remark, id)
	if err != nil {
		return Rule{}, err
	}
	if in.ListenPort != nil && *in.ListenPort != oldListenPort && in.Hosts == nil {
		if _, err := s.querier().ExecContext(ctx, `
			UPDATE proxy_hosts SET listen_port = ? WHERE rule_id = ? AND listen_port = ?
		`, listenPort, id, oldListenPort); err != nil {
			return Rule{}, err
		}
	}
	if in.Hosts != nil {
		if err := s.ensureHostsAvailable(ctx, hosts, listenPort, id); err != nil {
			return Rule{}, err
		}
		if err := s.replaceHosts(ctx, id, listenPort, hosts); err != nil {
			return Rule{}, err
		}
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
		SELECT id, upstream, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, enabled, remark, created_at, updated_at
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.attachHosts(ctx, rules)
}

func validateCreateInput(in CreateInput) (string, []Host, error) {
	upstream, err := validate.Upstream(in.Upstream)
	if err != nil {
		return "", nil, err
	}
	if err := validate.ListenPort(in.ListenPort); err != nil {
		return "", nil, err
	}
	if !in.ListenIPv4 && !in.ListenIPv6 {
		return "", nil, fmt.Errorf("至少需要启用 IPv4 或 IPv6 监听")
	}
	hosts, err := parseHosts(in.Hosts, in.ListenPort)
	if err != nil {
		return "", nil, err
	}
	if len(hosts) == 0 {
		return "", nil, fmt.Errorf("至少需要一个前端域名")
	}
	return upstream, hosts, nil
}

func (s *Store) cleanupOrphanRules(ctx context.Context) error {
	_, err := s.querier().ExecContext(ctx, `
		DELETE FROM proxy_rules
		WHERE id NOT IN (SELECT rule_id FROM proxy_hosts)
	`)
	return err
}

func hostEffectivePort(host Host, ruleListenPort int) int {
	if host.ListenPort != nil && *host.ListenPort > 0 {
		return *host.ListenPort
	}
	return ruleListenPort
}

func parseHosts(raw []string, ruleListenPort int) ([]Host, error) {
	seen := make(map[string]struct{})
	hosts := make([]Host, 0, len(raw))
	for _, item := range raw {
		hostname, port, err := validate.FrontendAddress(item)
		if err != nil {
			return nil, err
		}
		effectivePort := ruleListenPort
		if port > 0 {
			effectivePort = port
		}
		key := fmt.Sprintf("%s:%d", strings.ToLower(hostname), effectivePort)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("前端域名 %s:%d 重复", hostname, effectivePort)
		}
		seen[key] = struct{}{}
		host := Host{Hostname: hostname}
		if port > 0 {
			host.ListenPort = &port
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

func (s *Store) ensureHostAvailable(ctx context.Context, hostname string, port int, excludeRuleID int64) error {
	var ownerID int64
	err := s.querier().QueryRowContext(ctx, `
		SELECT rule_id FROM proxy_hosts WHERE hostname = ? COLLATE NOCASE AND listen_port = ? LIMIT 1
	`, hostname, port).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if ownerID != excludeRuleID {
		return fmt.Errorf("前端域名 %s:%d 已被其他规则使用", hostname, port)
	}
	return nil
}

func (s *Store) ensureHostsAvailable(ctx context.Context, hosts []Host, ruleListenPort int, excludeRuleID int64) error {
	for _, host := range hosts {
		port := hostEffectivePort(host, ruleListenPort)
		if err := s.ensureHostAvailable(ctx, host.Hostname, port, excludeRuleID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) replaceHosts(ctx context.Context, ruleID int64, ruleListenPort int, hosts []Host) error {
	if _, err := s.querier().ExecContext(ctx, `DELETE FROM proxy_hosts WHERE rule_id = ?`, ruleID); err != nil {
		return err
	}
	for _, host := range hosts {
		port := hostEffectivePort(host, ruleListenPort)
		_, err := s.querier().ExecContext(ctx, `
			INSERT INTO proxy_hosts(rule_id, hostname, listen_port)
			VALUES (?, ?, ?)
		`, ruleID, host.Hostname, port)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("前端域名 %s:%d 已被其他规则使用", host.Hostname, port)
			}
			return err
		}
	}
	return nil
}

func (s *Store) attachHosts(ctx context.Context, rules []Rule) ([]Rule, error) {
	if len(rules) == 0 {
		return rules, nil
	}
	ids := make([]string, len(rules))
	args := make([]any, len(rules))
	for i, rule := range rules {
		ids[i] = "?"
		args[i] = rule.ID
	}

	query := `
		SELECT id, rule_id, hostname, listen_port
		FROM proxy_hosts
		WHERE rule_id IN (` + strings.Join(ids, ",") + `)
		ORDER BY id ASC
	`
	rows, err := s.querier().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byRule := make(map[int64][]Host)
	for rows.Next() {
		var host Host
		var ruleID int64
		var listenPort sql.NullInt64
		if err := rows.Scan(&host.ID, &ruleID, &host.Hostname, &listenPort); err != nil {
			return nil, err
		}
		if listenPort.Valid {
			port := int(listenPort.Int64)
			host.ListenPort = &port
		}
		byRule[ruleID] = append(byRule[ruleID], host)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range rules {
		rules[i].Hosts = byRule[rules[i].ID]
		rules[i].Domain = rules[i].PrimaryHost()
	}
	return rules, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRule(row rowScanner) (Rule, error) {
	var rule Rule
	var listenIPv4 int
	var listenIPv6 int
	var httpsEnabled int
	var httpRedirect int
	var enabled int
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&rule.ID,
		&rule.Upstream,
		&rule.ListenPort,
		&listenIPv4,
		&listenIPv6,
		&httpsEnabled,
		&httpRedirect,
		&enabled,
		&rule.Remark,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Rule{}, err
	}
	rule.ListenIPv4 = listenIPv4 == 1
	rule.ListenIPv6 = listenIPv6 == 1
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

func normalizeRemark(raw string) (string, error) {
	remark := strings.TrimSpace(raw)
	if len(remark) > 100 {
		return "", fmt.Errorf("备注不能超过 100 个字符")
	}
	return remark, nil
}
