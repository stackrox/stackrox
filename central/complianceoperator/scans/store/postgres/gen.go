package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScan --postgres-migration-seq 12 --migrate-from "rocksdb:complianceoperatorscans"
