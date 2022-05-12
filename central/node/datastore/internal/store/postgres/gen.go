package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Node --table=nodes --search-category NODES --references=clusters:storage.Cluster --schema-only
