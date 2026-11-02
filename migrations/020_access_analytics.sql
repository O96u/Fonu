CREATE TABLE IF NOT EXISTS access_facts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts TEXT NOT NULL,
	client_ip TEXT NOT NULL,
	host TEXT NOT NULL DEFAULT '',
	service TEXT NOT NULL DEFAULT '',
	method TEXT NOT NULL DEFAULT '',
	path TEXT NOT NULL DEFAULT '',
	status INTEGER NOT NULL DEFAULT 0,
	bytes_sent INTEGER NOT NULL DEFAULT 0,
	request_ms REAL NOT NULL DEFAULT 0,
	user_agent TEXT NOT NULL DEFAULT '',
	kind TEXT NOT NULL DEFAULT 'access',
	geo_country TEXT NOT NULL DEFAULT '',
	geo_province TEXT NOT NULL DEFAULT '',
	geo_city TEXT NOT NULL DEFAULT '',
	geo_label TEXT NOT NULL DEFAULT '',
	geo_pending INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_access_facts_ts ON access_facts(ts);
CREATE INDEX IF NOT EXISTS idx_access_facts_client_ip ON access_facts(client_ip);
CREATE INDEX IF NOT EXISTS idx_access_facts_status ON access_facts(status);
CREATE INDEX IF NOT EXISTS idx_access_facts_geo_province ON access_facts(geo_province);
CREATE INDEX IF NOT EXISTS idx_access_facts_geo_pending ON access_facts(geo_pending);

CREATE TABLE IF NOT EXISTS ingest_state (
	log_path TEXT PRIMARY KEY,
	file_offset INTEGER NOT NULL DEFAULT 0,
	file_size INTEGER NOT NULL DEFAULT 0,
	file_mtime_ns INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS geo_cache (
	ip TEXT PRIMARY KEY,
	geo_country TEXT NOT NULL DEFAULT '',
	geo_province TEXT NOT NULL DEFAULT '',
	geo_city TEXT NOT NULL DEFAULT '',
	geo_label TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
