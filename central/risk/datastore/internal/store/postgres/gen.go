package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Risk --search-category RISKS --postgres-migration-seq 47 --migrate-from "rocksdb:risk"
