package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Secret --search-category SECRETS --migration-seq 47 --migrate-from rocksdb
