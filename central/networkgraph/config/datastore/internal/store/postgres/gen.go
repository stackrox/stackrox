package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkGraphConfig --postgres-migration-seq 31 --migrate-from "rocksdb:networkgraphconfig"
