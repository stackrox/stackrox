package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkGraphConfig --migration-seq 31 --migrate-from rocksdb
