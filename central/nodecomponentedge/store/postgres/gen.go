package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeComponentEdge --search-category NODE_COMPONENT_EDGE --references=storage.Node,node_components:storage.NodeComponent --read-only-store --search-scope NODE_VULNERABILITIES,NODE_COMPONENT_CVE_EDGE,NODE_COMPONENTS,NODE_COMPONENT_EDGE,NODES,CLUSTERS
