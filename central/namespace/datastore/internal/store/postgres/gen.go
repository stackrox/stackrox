package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NamespaceMetadata --cached-store --for-sac --table=namespaces --search-category NAMESPACES --references=storage.Cluster --search-scope IMAGE_VULNERABILITIES_V2,IMAGE_COMPONENTS_V2,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS --default-sort search.Namespace.String()
