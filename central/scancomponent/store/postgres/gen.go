package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ScanComponent --schema-only --references=imageScanV2:storage.ImageScanV2
