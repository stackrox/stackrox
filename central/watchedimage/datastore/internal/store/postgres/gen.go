package postgres

//go:generate pg-table-bindings-wrapper --type=storage.WatchedImage --migration-seq 54 --migrate-from rocksdb
