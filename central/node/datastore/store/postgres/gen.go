package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Node --search-category NODES --references=storage.Cluster --schema-only --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
