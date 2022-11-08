package postgres

//go:generate pg-table-bindings-wrapper --type=storage.IntegrationHealth --migration-seq 25 --migrate-from rocksdb
