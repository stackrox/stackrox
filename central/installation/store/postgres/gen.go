package postgres

//go:generate pg-table-bindings-wrapper --type=storage.InstallationInfo --singleton --postgres-migration-seq 24 --migrate-from "boltdb:installationInfo"
