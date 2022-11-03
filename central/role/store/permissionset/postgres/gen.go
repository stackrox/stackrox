package postgres

//go:generate pg-table-bindings-wrapper --type=storage.PermissionSet
// To regenerate migration, add:
// --migration-seq 37 --migrate-from rocksdb
