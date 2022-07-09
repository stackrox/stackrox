package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ExternalBackup --get-all-func --postgres-migration-seq 22 --migrate-from "boltdb:externalBackups"
