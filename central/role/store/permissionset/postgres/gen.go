package postgres

//go:generate pg-table-bindings-wrapper --type=storage.PermissionSet --postgres-migration-seq 40 --migrate-from "rocksdb:permission_sets"
