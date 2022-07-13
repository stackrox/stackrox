package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageIntegration --get-all-func --migration-seq 23 --migrate-from boltdb
