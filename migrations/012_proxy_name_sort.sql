ALTER TABLE proxy_rules RENAME COLUMN remark TO name;

ALTER TABLE proxy_rules ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

UPDATE proxy_rules SET sort_order = id;

CREATE INDEX IF NOT EXISTS idx_proxy_rules_sort_order ON proxy_rules(sort_order);
