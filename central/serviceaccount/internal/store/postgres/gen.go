package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ServiceAccount --search-category SERVICE_ACCOUNTS --migration-seq 49 --migrate-from rocksdb
