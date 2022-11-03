package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Role --bucket=roles --cache --key-func GetName()
// To regenerate migration, add:
// --migration-seq 46 --migrate-to roles
