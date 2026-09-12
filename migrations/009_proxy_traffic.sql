CREATE TABLE IF NOT EXISTS proxy_traffic (
    host TEXT PRIMARY KEY,
    upload_bytes INTEGER NOT NULL DEFAULT 0,
    download_bytes INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
