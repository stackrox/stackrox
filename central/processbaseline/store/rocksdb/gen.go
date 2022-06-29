package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ProcessBaseline --bucket=processWhitelists2 --cache --migrate-seq 44 --migrate-to process_baselines
