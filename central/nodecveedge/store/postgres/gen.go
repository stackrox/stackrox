package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeCVEEdge --table=nodes_to_cves --search-category NODE_VULN_EDGE --references=nodes:storage.Node,node_cves:storage.CVE --join-table true
