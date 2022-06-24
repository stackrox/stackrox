package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Policy --search-category POLICIES

// To generate again, add the following parameters.
// --postgres-migration-seq 42 --migrate-from "boltdb:removed_default_policies"
