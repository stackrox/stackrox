package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageV2 --table=images_v2 --search-category IMAGES_V2 --schema-only --search-scope IMAGE_VULNERABILITIES_V2,IMAGE_COMPONENTS_V2,IMAGES_V2,DEPLOYMENTS,NAMESPACES,CLUSTERS --feature-flag FlattenImageData
