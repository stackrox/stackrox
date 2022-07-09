package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Alert --search-category ALERTS --postgres-migration-seq 6 --migrate-from "rocksdb:alerts"
