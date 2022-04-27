package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Deployment --table=deployments --search-category SearchCategory_DEPLOYMENTS --references=images:storage.Image
