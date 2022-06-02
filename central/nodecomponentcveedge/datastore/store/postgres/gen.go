package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdge --table=node_components_cves_edges --search-category NODE_COMPONENT_CVE_EDGE --references=node_components:storage.ImageComponent,node_cves:storage.CVE --join-table true
