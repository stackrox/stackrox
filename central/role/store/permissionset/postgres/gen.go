package postgres

//go:generate pg-table-bindings-wrapper --type=storage.PermissionSet --migration-seq 37 --migrate-from rocksdb
