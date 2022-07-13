package rocksdb

//go:generate rocksdb-bindings-wrapper --type=PermissionSet --bucket=permission_sets --cache --uniq-key-func GetName() --migration-seq 37 --migrate-to permission_sets
