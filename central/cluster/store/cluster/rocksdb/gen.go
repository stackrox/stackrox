package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Cluster --bucket=clusters --cache --uniq-key-func GetName() --migrate-seq 5 --migrate-to clusters
