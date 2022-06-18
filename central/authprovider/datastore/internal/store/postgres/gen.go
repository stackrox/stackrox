package postgres

//go:generate pg-table-bindings-wrapper --type=storage.AuthProvider --get-all-func --postgres-migration-seq 3 --migrate-from "boltdb:authProviders"
