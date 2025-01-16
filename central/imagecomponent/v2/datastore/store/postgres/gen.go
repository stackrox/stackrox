package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentV2 --references=storage.Image --search-category IMAGE_COMPONENTS_V2 --search-scope IMAGE_COMPONENTS_V2,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
