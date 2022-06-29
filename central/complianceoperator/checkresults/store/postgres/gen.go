package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceOperatorCheckResult --postgres-migration-seq 8 --migrate-from "rocksdb:complianceoperatorcheckresults"
