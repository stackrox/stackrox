package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkPolicy --table=networkpolicies --postgres-migration-seq 31 --migrate-from "boltdb:networkpolicies"
