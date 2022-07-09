package postgres

//go:generate pg-table-bindings-wrapper --type=storage.PolicyCategory --search-category POLICY_CATEGORIES --postgres-migration-seq 55 --migrate-from "rocksdb:policy_categories"
