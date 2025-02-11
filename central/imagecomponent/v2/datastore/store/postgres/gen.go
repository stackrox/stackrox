package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentV2 --references=storage.Image --search-category IMAGE_COMPONENTS --search-scope IMAGE_VULNERABILITIES,IMAGE_COMPONENTS,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS --feature-flag FlattenCVEData
