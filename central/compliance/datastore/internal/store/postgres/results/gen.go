package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceRunResults --search-category COMPLIANCE_RESULTS --migration-seq 19 --migrate-from rocksdb
