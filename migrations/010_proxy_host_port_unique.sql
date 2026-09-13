UPDATE proxy_hosts
SET listen_port = (
    SELECT listen_port FROM proxy_rules WHERE proxy_rules.id = proxy_hosts.rule_id
)
WHERE listen_port IS NULL;

CREATE TABLE proxy_hosts_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL REFERENCES proxy_rules(id) ON DELETE CASCADE,
    hostname TEXT NOT NULL COLLATE NOCASE,
    listen_port INTEGER NOT NULL CHECK (listen_port > 0 AND listen_port <= 65535),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(hostname, listen_port)
);

INSERT INTO proxy_hosts_new (id, rule_id, hostname, listen_port, created_at)
SELECT id, rule_id, hostname, listen_port, created_at FROM proxy_hosts;

DROP TABLE proxy_hosts;
ALTER TABLE proxy_hosts_new RENAME TO proxy_hosts;

CREATE INDEX IF NOT EXISTS idx_proxy_hosts_rule_id ON proxy_hosts(rule_id);
