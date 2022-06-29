package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Role --postgres-migration-seq 49 --migrate-from "rocksdb:roles"
