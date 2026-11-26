CREATE TABLE IF NOT EXISTS proxy_business_groups (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL COLLATE NOCASE UNIQUE,
 sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS proxy_business_members (
 rule_id INTEGER PRIMARY KEY REFERENCES proxy_rules(id) ON DELETE CASCADE,
 group_id INTEGER NOT NULL REFERENCES proxy_business_groups(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_proxy_business_members_group ON proxy_business_members(group_id);
