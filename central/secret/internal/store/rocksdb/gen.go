package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Secret --bucket=secrets --migrate-seq 50 --migrate-to secrets
