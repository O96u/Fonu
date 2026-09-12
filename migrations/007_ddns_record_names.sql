ALTER TABLE ddns_configs ADD COLUMN record_names TEXT NOT NULL DEFAULT '[]';

UPDATE ddns_configs
SET record_names = json_array(record_name)
WHERE record_names = '[]' OR record_names IS NULL OR record_names = '';
