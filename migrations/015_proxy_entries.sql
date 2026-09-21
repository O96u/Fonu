CREATE TABLE IF NOT EXISTS proxy_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL DEFAULT '',
    listen_port INTEGER NOT NULL,
    listen_ipv4 INTEGER NOT NULL DEFAULT 1 CHECK (listen_ipv4 IN (0, 1)),
    listen_ipv6 INTEGER NOT NULL DEFAULT 0 CHECK (listen_ipv6 IN (0, 1)),
    https_enabled INTEGER NOT NULL DEFAULT 1 CHECK (https_enabled IN (0, 1)),
    http_redirect INTEGER NOT NULL DEFAULT 1 CHECK (http_redirect IN (0, 1)),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

ALTER TABLE proxy_rules ADD COLUMN entry_id INTEGER REFERENCES proxy_entries(id) ON DELETE SET NULL;

INSERT INTO proxy_entries (name, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, sort_order, updated_at)
SELECT
    '',
    listen_port,
    listen_ipv4,
    listen_ipv6,
    https_enabled,
    http_redirect,
    listen_port,
    datetime('now')
FROM proxy_rules
GROUP BY listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect;

UPDATE proxy_rules
SET entry_id = (
    SELECT e.id
    FROM proxy_entries e
    WHERE e.listen_port = proxy_rules.listen_port
      AND e.listen_ipv4 = proxy_rules.listen_ipv4
      AND e.listen_ipv6 = proxy_rules.listen_ipv6
      AND e.https_enabled = proxy_rules.https_enabled
      AND e.http_redirect = proxy_rules.http_redirect
    LIMIT 1
);

CREATE INDEX IF NOT EXISTS idx_proxy_rules_entry_id ON proxy_rules(entry_id);
CREATE INDEX IF NOT EXISTS idx_proxy_entries_sort_order ON proxy_entries(sort_order);
