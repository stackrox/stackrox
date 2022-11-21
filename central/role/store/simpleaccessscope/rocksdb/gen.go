package rocksdb

//go:generate rocksdb-bindings-wrapper --type=SimpleAccessScope --bucket=simple_access_scopes --cache --uniq-key-func GetName()
