package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdge --search-category COMPONENT_VULN_EDGE --references=storage.ImageComponent,image_cves:storage.CVE --join-table true
