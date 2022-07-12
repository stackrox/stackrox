package rocksdb

//go:generate rocksdb-bindings-wrapper --type=SimpleAccessScope --bucket=simple_access_scopes --cache --uniq-key-func GetName() --migration-seq 52 --migrate-to simple_access_scopes
