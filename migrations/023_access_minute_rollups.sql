-- Exact per-minute per-IP counters; retain raw events for boundary and drill-down queries.
CREATE TABLE IF NOT EXISTS access_minute_ip (
    minute TEXT NOT NULL,
    client_ip TEXT NOT NULL,
    requests INTEGER NOT NULL,
    bytes INTEGER NOT NULL,
    PRIMARY KEY(minute,client_ip)
) WITHOUT ROWID;

INSERT INTO access_minute_ip(minute,client_ip,requests,bytes)
SELECT substr(ts,1,16)||':00Z',client_ip,COUNT(*),COALESCE(SUM(bytes_sent),0)
FROM access_facts GROUP BY 1,2;

CREATE TRIGGER access_minute_ip_insert AFTER INSERT ON access_facts BEGIN
    INSERT INTO access_minute_ip(minute,client_ip,requests,bytes)
    VALUES(substr(NEW.ts,1,16)||':00Z',NEW.client_ip,1,NEW.bytes_sent)
    ON CONFLICT(minute,client_ip) DO UPDATE SET requests=requests+1,bytes=bytes+NEW.bytes_sent;
END;

CREATE TRIGGER access_minute_ip_delete AFTER DELETE ON access_facts BEGIN
    UPDATE access_minute_ip SET requests=requests-1,bytes=bytes-OLD.bytes_sent
    WHERE minute=substr(OLD.ts,1,16)||':00Z' AND client_ip=OLD.client_ip;
    DELETE FROM access_minute_ip WHERE minute=substr(OLD.ts,1,16)||':00Z' AND client_ip=OLD.client_ip AND requests<=0;
END;

CREATE TRIGGER access_minute_ip_update AFTER UPDATE OF ts,client_ip,bytes_sent ON access_facts BEGIN
    UPDATE access_minute_ip SET requests=requests-1,bytes=bytes-OLD.bytes_sent
    WHERE minute=substr(OLD.ts,1,16)||':00Z' AND client_ip=OLD.client_ip;
    DELETE FROM access_minute_ip WHERE minute=substr(OLD.ts,1,16)||':00Z' AND client_ip=OLD.client_ip AND requests<=0;
    INSERT INTO access_minute_ip(minute,client_ip,requests,bytes)
    VALUES(substr(NEW.ts,1,16)||':00Z',NEW.client_ip,1,NEW.bytes_sent)
    ON CONFLICT(minute,client_ip) DO UPDATE SET requests=requests+1,bytes=bytes+NEW.bytes_sent;
END;
