package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentV3 --references=storage.Image --search-category IMAGE_COMPONENTS_V3 --search-scope IMAGE_VULNERABILITIES_V3,IMAGE_COMPONENTS_V3,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
