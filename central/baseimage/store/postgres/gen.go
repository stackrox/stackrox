package postgres

//go:generate pg-table-bindings-wrapper --type=storage.BaseImage --table=base_images --search-category=BASE_IMAGES --search-scope=BASE_IMAGES,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
