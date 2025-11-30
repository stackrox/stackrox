package postgres

//go:generate pg-table-bindings-wrapper --type=storage.BaseImageRepository --table=base_image_repository --search-category=BASE_IMAGE_REPOSITORIES --search-scope=BASE_IMAGE_REPOSITORIES, BASE_IMAGES,IMAGES,IMAGES_V2,DEPLOYMENTS,NAMESPACES,CLUSTERS
