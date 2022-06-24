package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Notifier --get-all-func --postgres-migration-seq 39 --migrate-from "boltdb:notifiers"
