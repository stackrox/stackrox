package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Role --bucket=roles --cache --key-func GetName() --migration-seq 46 --migrate-to roles
