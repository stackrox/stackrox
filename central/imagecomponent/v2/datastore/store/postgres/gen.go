package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentV2 --references=storage.Image,imagesV2:storage.ImageV2 --search-category IMAGE_COMPONENTS_V2 --search-scope IMAGE_VULNERABILITIES_V2,IMAGE_COMPONENTS_V2,IMAGES,IMAGES_V2,DEPLOYMENTS,NAMESPACES,CLUSTERS --feature-flag FlattenCVEData --default-sort search.Component.String() --transform-sort-options ImagesSchema.OptionsMap
