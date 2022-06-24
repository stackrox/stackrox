package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkBaseline --search-category NETWORK_BASELINE --postgres-migration-seq 27 --migrate-from "rocksdb:networkbaseline"
