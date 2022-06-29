package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Config --singleton --postgres-migration-seq 13 --migrate-from "boltdb:config"
