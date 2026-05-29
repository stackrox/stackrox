package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ScanFinding --schema-only --references=scanComponents:storage.ScanComponent,imageScanV2:storage.ImageScanV2,imagesV2:storage.ImageV2
