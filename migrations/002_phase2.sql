CREATE TABLE IF NOT EXISTS ddns_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL DEFAULT 'cloudflare',
    root_domain TEXT NOT NULL,
    record_name TEXT NOT NULL,
    ipv4_enabled INTEGER NOT NULL DEFAULT 1 CHECK (ipv4_enabled IN (0, 1)),
    ipv6_enabled INTEGER NOT NULL DEFAULT 0 CHECK (ipv6_enabled IN (0, 1)),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    api_token_enc TEXT,
    last_ipv4 TEXT,
    last_ipv6 TEXT,
    last_status TEXT NOT NULL DEFAULT 'unknown',
    last_error TEXT,
    last_updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL UNIQUE,
    wildcard INTEGER NOT NULL DEFAULT 0 CHECK (wildcard IN (0, 1)),
    cert_path TEXT,
    key_path TEXT,
    expires_at TEXT,
    last_renew_at TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    last_error TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
