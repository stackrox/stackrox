package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageCVEEdge --references=storage.Image,image_cves:storage.ImageCVE --schema-only
