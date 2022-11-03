package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Role
// To regenerate migration, add:
// --migration-seq 46 --migrate-from rocksdb
