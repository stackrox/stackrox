ALTER TABLE last_vuln_update
    ADD COLUMN update_timestamp TIMESTAMP
        DEFAULT to_timestamp(0)
        NOT NULL;

UPDATE last_vuln_update
    SET update_timestamp = TO_TIMESTAMP(timestamp, 'Dy, DD Mon YYYY HH24:MI:SS')
    WHERE TRIM(timestamp) <> '';
