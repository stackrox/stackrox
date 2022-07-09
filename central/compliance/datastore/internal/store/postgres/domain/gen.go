package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComplianceDomain --search-category COMPLIANCE_DOMAIN --postgres-migration-seq 12 --migrate-from "rocksdb:unknown"
