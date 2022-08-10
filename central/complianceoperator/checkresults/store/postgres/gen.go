package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorCheckResult --migration-seq 13 --migrate-from rocksdb --references storage.Cluster
