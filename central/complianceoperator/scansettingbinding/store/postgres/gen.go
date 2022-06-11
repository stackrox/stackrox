package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScanSettingBinding --postgres-migration-seq 11 --migrate-from "rocksdb:complianceoperatorscansettingbinding"
