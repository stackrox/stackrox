package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceStrings --postgres-migration-seq 20 --migrate-from "rocksdb:unknown"
