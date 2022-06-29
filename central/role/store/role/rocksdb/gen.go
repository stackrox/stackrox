package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Role --bucket=roles --cache --key-func GetName() --migrate-seq 49 --migrate-to roles
