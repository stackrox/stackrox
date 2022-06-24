package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeComponentCVEEdge --table=node_components_cves_edges --search-category NODE_COMPONENT_CVE_EDGE --references=node_components:storage.NodeComponent,node_cves:storage.NodeCVE --join-table true
