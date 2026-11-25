ALTER TABLE certificates ADD COLUMN auto_renew INTEGER NOT NULL DEFAULT 1;
UPDATE certificates SET auto_renew = 0 WHERE acme_ca = 'imported';
