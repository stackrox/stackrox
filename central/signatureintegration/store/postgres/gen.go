package postgres

//go:generate pg-table-bindings-wrapper --type=storage.SignatureIntegration --postgres-migration-seq 51 --migrate-from "rocksdb:signature_integrations"
