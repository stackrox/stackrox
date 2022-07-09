package rocksdb

//go:generate rocksdb-bindings-wrapper --type=TokenMetadata --bucket=apiTokens --migrate-seq 7 --migrate-to api_tokens
