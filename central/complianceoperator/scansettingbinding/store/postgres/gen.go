package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScanSettingBinding --migration-seq 16 --migrate-from rocksdb
