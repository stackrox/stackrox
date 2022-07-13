package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessBaselineResults --search-category PROCESS_BASELINE_RESULTS --migration-seq 40 --migrate-from rocksdb
