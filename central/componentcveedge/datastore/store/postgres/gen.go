package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdge --table=image_component_cve_edges --references=storage.ImageComponent,image_cves:storage.ImageCVE --schema-only
