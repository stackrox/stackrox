package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Deployment --search-category DEPLOYMENTS --references=storage.Image,namespaces:storage.NamespaceMetadata
