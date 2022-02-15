package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Deployment --table=deployments --write-options=false --options-path mappings --singular Deployment --search-category DEPLOYMENTS
