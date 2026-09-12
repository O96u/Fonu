CREATE UNIQUE INDEX IF NOT EXISTS idx_ddns_configs_domain_record
ON ddns_configs(root_domain, record_name);
