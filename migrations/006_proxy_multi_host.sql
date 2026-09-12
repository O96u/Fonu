CREATE TEMP TABLE proxy_domain_backup AS
SELECT id, domain FROM proxy_rules;

CREATE TABLE proxy_rules_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    upstream TEXT NOT NULL,
    listen_port INTEGER NOT NULL DEFAULT 80 CHECK (listen_port > 0 AND listen_port <= 65535),
    listen_ipv4 INTEGER NOT NULL DEFAULT 1 CHECK (listen_ipv4 IN (0, 1)),
    listen_ipv6 INTEGER NOT NULL DEFAULT 0 CHECK (listen_ipv6 IN (0, 1)),
    https_enabled INTEGER NOT NULL DEFAULT 1 CHECK (https_enabled IN (0, 1)),
    http_redirect INTEGER NOT NULL DEFAULT 1 CHECK (http_redirect IN (0, 1)),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO proxy_rules_new (id, upstream, listen_port, listen_ipv4, listen_ipv6, https_enabled, http_redirect, enabled, created_at, updated_at)
SELECT
    id,
    upstream,
    CASE WHEN https_enabled = 1 THEN 443 ELSE 80 END,
    1,
    0,
    https_enabled,
    http_redirect,
    enabled,
    created_at,
    updated_at
FROM proxy_rules;

DROP TABLE proxy_rules;
ALTER TABLE proxy_rules_new RENAME TO proxy_rules;

CREATE TABLE IF NOT EXISTS proxy_hosts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL REFERENCES proxy_rules(id) ON DELETE CASCADE,
    hostname TEXT NOT NULL COLLATE NOCASE,
    listen_port INTEGER CHECK (listen_port IS NULL OR (listen_port > 0 AND listen_port <= 65535)),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(hostname)
);

CREATE INDEX IF NOT EXISTS idx_proxy_hosts_rule_id ON proxy_hosts(rule_id);

INSERT INTO proxy_hosts (rule_id, hostname, listen_port)
SELECT id, domain, NULL
FROM proxy_domain_backup
WHERE domain IS NOT NULL AND TRIM(domain) != '';

DROP TABLE proxy_domain_backup;
