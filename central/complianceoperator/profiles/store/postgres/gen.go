package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorProfile --postgres-migration-seq 14 --migrate-from "rocksdb:complianceoperatorprofiles"
