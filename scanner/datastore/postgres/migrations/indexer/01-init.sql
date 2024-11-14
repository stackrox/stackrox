--- a table to hold deletion timestamps for manifests.
CREATE TABLE IF NOT EXISTS manifest_metadata (
    manifest_id TEXT PRIMARY KEY,
    expiration  TIMESTAMP
);

--- (hopefully) make 'WHERE expiration < SOME_TIMESTAMP' searches faster.
CREATE INDEX IF NOT EXISTS manifest_metadata_expiration_idx ON manifest_metadata(expiration);
