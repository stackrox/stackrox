package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Cluster --bucket=clusters --cache --uniq-key-func GetName()
