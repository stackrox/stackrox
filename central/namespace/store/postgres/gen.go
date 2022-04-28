package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NamespaceMetadata --table=namespaces --search-category NAMESPACES --references=clusters:storage.Cluster
