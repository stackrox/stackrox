package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessIndicator --search-category PROCESS_INDICATORS --references storage.Deployment
