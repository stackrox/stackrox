package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TokenMetadata --table=api_tokens --postgres-migration-seq 7 --migrate-from "rocksdb:apiTokens"
