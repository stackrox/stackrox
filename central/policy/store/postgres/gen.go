package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Policy --search-category POLICIES --postgres-migration-seq 42 --migrate-from "boltdb:removed_default_policies"
