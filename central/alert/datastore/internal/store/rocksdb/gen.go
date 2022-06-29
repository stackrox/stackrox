package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Alert --bucket=alerts --track-index --migrate-seq 1 --migrate-to alerts
