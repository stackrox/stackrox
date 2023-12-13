--- a table to hold the timestamp for the latest vulnerability update.
CREATE TABLE IF NOT EXISTS last_vuln_update (
    id SERIAL PRIMARY KEY,
    key VARCHAR(128) NOT NULL UNIQUE,
    timestamp TEXT
);
