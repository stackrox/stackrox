package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Pod --search-category PODS --references storage.Deployment --migration-seq 38 --migrate-from rocksdb
