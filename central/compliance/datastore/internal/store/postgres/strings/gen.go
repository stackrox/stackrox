package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceStrings --migration-seq 20 --migrate-from rocksdb
