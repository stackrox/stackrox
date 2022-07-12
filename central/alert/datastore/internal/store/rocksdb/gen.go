package rocksdb

//go:generate rocksdb-bindings-wrapper --type=Alert --bucket=alerts --track-index --migration-seq 6 --migrate-to alerts
