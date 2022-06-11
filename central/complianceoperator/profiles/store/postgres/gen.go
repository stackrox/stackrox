package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorProfile --postgres-migration-seq 9 --migrate-from "rocksdb:complianceoperatorprofiles"
