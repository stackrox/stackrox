package postgres

//go:generate pg-table-bindings-wrapper --type=storage.BaseImageLayer --table=base_image_layers --search-category=BASE_IMAGE_LAYERS --search-scope=BASE_IMAGES,BASE_IMAGE_LAYERS --references=base_images:storage.BaseImage
