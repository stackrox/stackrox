package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NormalizedComponentCVEEdge --table=component_cve_edges --search-category IMAGE_COMPONENT_CVE_EDGE --references=image_component_v2:storage.ImageComponentV2,cves:storage.NormalizedCVE
