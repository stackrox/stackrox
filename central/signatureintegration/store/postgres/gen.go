package postgres

//go:generate pg-table-bindings-wrapper --type=storage.SignatureIntegration --postgres-migration-seq 54 --migrate-from "rocksdb:signature_integrations"
