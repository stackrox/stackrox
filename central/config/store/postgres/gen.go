package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Config --singleton --postgres-migration-seq 21 --migrate-from "boltdb:config"
