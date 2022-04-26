package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdge --table=image_component_cve_relations --search-category COMPONENT_VULN_EDGE --options-path central/componentcveedge/mappings  --references=image_components:storage.ImageComponent,image_cves:storage.CVE --join-table true
