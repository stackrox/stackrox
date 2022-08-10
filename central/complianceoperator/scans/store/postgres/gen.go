package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorScan --migration-seq 17 --migrate-from rocksdb --references storage.Cluster
