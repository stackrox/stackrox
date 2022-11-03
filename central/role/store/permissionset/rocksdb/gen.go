package rocksdb

//go:generate rocksdb-bindings-wrapper --type=PermissionSet --bucket=permission_sets --cache --uniq-key-func GetName()
// To regenerate migration, add:
// --migration-seq 37 --migrate-to permission_sets
