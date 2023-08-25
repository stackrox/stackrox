package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeComponent --table=node_components --search-category NODE_COMPONENTS --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
