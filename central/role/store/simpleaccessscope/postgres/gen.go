package postgres

//go:generate pg-table-bindings-wrapper --type=storage.SimpleAccessScope --migration-seq 52 --migrate-from rocksdb
