package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ReportConfiguration --search-category REPORT_CONFIGURATIONS --postgres-migration-seq 46 --migrate-from "rocksdb:report_configs"
