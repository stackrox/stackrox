package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Cluster --bucket=clusters --cache --uniq-key-func GetName() --migration-seq 1 --migrate-to clusters
