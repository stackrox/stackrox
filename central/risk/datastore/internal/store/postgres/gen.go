package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Risk --search-category RISKS --migration-seq 44 --migrate-from rocksdb
