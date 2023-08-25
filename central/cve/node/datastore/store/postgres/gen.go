package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeCVE --table=node_cves --search-category NODE_VULNERABILITIES --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
