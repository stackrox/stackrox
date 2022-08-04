package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ReportConfiguration --search-category REPORT_CONFIGURATIONS --migration-seq 43 --migrate-from rocksdb
