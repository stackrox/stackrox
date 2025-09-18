--- the jsonb serialized result of an index report from a delegated scan.
CREATE TABLE IF NOT EXISTS external_index_report (
    hash_id TEXT PRIMARY KEY,
    indexer_version TEXT NOT NULL,
    index_report JSONB NOT NULL,
    expiration TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS external_index_report_expiration_idx ON external_index_report(expiration);
