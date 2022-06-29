package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorRule --postgres-migration-seq 10 --migrate-from "rocksdb:complianceoperatorrules"
