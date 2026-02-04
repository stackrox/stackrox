package postgres

//go:generate pg-table-bindings-wrapper --type=storage.BaseImage --references=storage.BaseImageRepository,storage.Image,imagesV2:storage.ImageV2 --table=base_images --search-category=BASE_IMAGES --search-scope=BASE_IMAGES,IMAGES,IMAGES_V2,DEPLOYMENTS,NAMESPACES,CLUSTERS
