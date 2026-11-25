-- Legacy bearer tokens must not remain in SQLite after upgrade.
DELETE FROM sessions WHERE id NOT LIKE 'sha256:%';
