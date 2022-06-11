package rocksdb

//go:generate rocksdb-bindings-wrapper --type=SimpleAccessScope --bucket=simple_access_scopes --cache --uniq-key-func GetName() --migrate-seq 55 --migrate-to simple_access_scopes
