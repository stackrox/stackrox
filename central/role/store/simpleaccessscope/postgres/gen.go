package postgres

//go:generate pg-table-bindings-wrapper --type=storage.SimpleAccessScope
// To regenerate migration, add:
// --migration-seq 52 --migrate-from rocksdb
