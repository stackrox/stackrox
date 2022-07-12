package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ProcessBaseline --bucket=processWhitelists2 --cache --migration-seq 41 --migrate-to process_baselines
