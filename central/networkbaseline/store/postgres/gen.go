package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkBaseline --postgres-migration-seq 27 --migrate-from "rocksdb:networkbaseline"
