package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Alert --bucket=alerts --track-index --migrate-seq 6 --migrate-to alerts
