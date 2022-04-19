package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NamespaceMetadata --table=namespaces --search-category NAMESPACES --options-path central/namespace/index/mappings --references=clusters:storage.Cluster
