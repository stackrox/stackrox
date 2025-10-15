package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Image --search-category IMAGES --schema-only --search-scope IMAGE_VULNERABILITIES_V2,IMAGE_COMPONENTS_V2,DEPLOYMENTS,NAMESPACES,CLUSTERS
