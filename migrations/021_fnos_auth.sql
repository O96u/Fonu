CREATE TABLE IF NOT EXISTS fnos_auth_bindings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nas_uid TEXT NOT NULL UNIQUE,
    nas_username TEXT NOT NULL DEFAULT '',
    fonu_admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

ALTER TABLE sessions ADD COLUMN auth_source TEXT NOT NULL DEFAULT 'password';
ALTER TABLE sessions ADD COLUMN nas_uid TEXT NOT NULL DEFAULT '';
