package rocksdb

//go:generate rocksdb-bindings-wrapper --type=TokenMetadata --bucket=apiTokens --migration-seq 7 --migrate-to api_tokens
