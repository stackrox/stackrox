package postgres

//go:generate pg-table-bindings-wrapper --type=storage.SignatureIntegration --migration-seq 51 --migrate-from rocksdb
