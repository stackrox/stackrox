package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Secret --search-category SECRETS --postgres-migration-seq 47 --migrate-from "rocksdb:secrets"
