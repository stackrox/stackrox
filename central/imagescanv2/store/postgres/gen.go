package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageScanV2 --schema-only --references=imagesV2:storage.ImageV2
