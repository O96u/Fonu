ALTER TABLE certificates ADD COLUMN dns_config_id INTEGER NOT NULL DEFAULT 0;
CREATE TABLE cert_deployment_history (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 domain TEXT NOT NULL,
 channel TEXT NOT NULL,
 operation TEXT NOT NULL,
 ok INTEGER NOT NULL,
 created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX idx_cert_deployment_domain ON cert_deployment_history(domain,id);
