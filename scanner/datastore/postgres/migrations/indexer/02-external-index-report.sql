--- the jsonb serialized result of an index report from a delegated scanner.
CREATE TABLE IF NOT EXISTS external_index_report (
    hash_id TEXT PRIMARY KEY,
    index_report JSONB NOT NULL,
    expiration TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS external_index_report_expiration_idx ON external_index_report(expiration);
