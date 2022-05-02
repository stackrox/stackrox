package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Deployment --table=deployments --search-category DEPLOYMENTS --references=images:storage.Image,namespaces:storage.NamespaceMetadata
