ALTER TABLE last_vuln_update
    ADD COLUMN IF NOT EXISTS update_timestamp TIMESTAMP DEFAULT to_timestamp(0) NOT NULL;
