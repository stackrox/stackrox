--- a table to hold the timestamp for the latest vulnerability update.
CREATE TABLE IF NOT EXISTS last_vuln_update (
    key       VARCHAR(128) PRIMARY KEY,
    timestamp TEXT
);
