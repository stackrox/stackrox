CREATE TABLE IF NOT EXISTS manifest_metadata (
    manifest_id TEXT PRIMARY KEY,
    expiration  TIMESTAMP
);
CREATE INDEX IF NOT EXISTS manifest_metadata_expiration_idx ON manifest_metadata(expiration);
