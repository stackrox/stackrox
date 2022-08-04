package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Notifier --get-all-func --migration-seq 36 --migrate-from boltdb
