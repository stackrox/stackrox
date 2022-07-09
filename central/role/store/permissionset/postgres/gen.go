package postgres

//go:generate pg-table-bindings-wrapper --type=storage.PermissionSet --postgres-migration-seq 37 --migrate-from "rocksdb:permission_sets"
