package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessBaselineResults --postgres-migration-seq 43 --migrate-from "rocksdb:processWhitelistResults"
