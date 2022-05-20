package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdge --table=image_component_cve_edges --search-category COMPONENT_VULN_EDGE --references=storage.ImageComponent,image_cves:storage.CVE --join-table true
