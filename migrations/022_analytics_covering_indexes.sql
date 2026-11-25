CREATE INDEX IF NOT EXISTS idx_access_facts_ts_status ON access_facts(ts, status);
CREATE INDEX IF NOT EXISTS idx_access_facts_pending_ip ON access_facts(client_ip) WHERE geo_pending = 1;
