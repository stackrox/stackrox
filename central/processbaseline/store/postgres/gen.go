package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessBaseline --search-category PROCESS_BASELINES --migration-seq 41 --migrate-from rocksdb
