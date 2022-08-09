package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Config --singleton --migration-seq 21 --migrate-from boltdb
