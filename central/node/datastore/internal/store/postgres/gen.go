package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Node --search-category NODES --references=storage.Cluster --schema-only
