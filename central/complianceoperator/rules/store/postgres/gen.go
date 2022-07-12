package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorRule --migration-seq 15 --migrate-from rocksdb
