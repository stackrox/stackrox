package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeComponentCVEEdge --search-category NODE_COMPONENT_CVE_EDGE --references=storage.ImageComponent,storage.CVE --join-table true
