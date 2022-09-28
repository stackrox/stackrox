package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Policy --search-category POLICIES --get-all-func
// To regenerate migration:
// --migration-seq 39 --migrate-from boltdb
