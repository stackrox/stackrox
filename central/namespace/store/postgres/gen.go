package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NamespaceMetadata --search-category NAMESPACES --references=storage.Cluster
