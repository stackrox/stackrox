package rocksdb

//go:generate rocksdb-bindings-wrapper --type=SimpleAccessScope --bucket=simple_access_scopes --cache --uniq-key-func GetName()
// To regenerate migration, add:
// --migration-seq 52 --migrate-to simple_access_scopes
