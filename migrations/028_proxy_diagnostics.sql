CREATE TABLE IF NOT EXISTS proxy_diagnostics (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 rule_id INTEGER NOT NULL REFERENCES proxy_rules(id) ON DELETE CASCADE,
 upstream TEXT NOT NULL,
 checked_at TEXT NOT NULL,
 state TEXT NOT NULL,
 duration_ms INTEGER NOT NULL,
 http_status INTEGER NOT NULL DEFAULT 0,
 detail TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_proxy_diagnostics_rule ON proxy_diagnostics(rule_id,id DESC);
