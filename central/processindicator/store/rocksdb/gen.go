package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ProcessIndicator --bucket=process_indicators2 --track-index --migrate-seq 42 --migrate-to process_indicators
