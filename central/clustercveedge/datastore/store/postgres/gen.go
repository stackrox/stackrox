package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ClusterCVEEdge --table=cluster_cve_edges --search-category CLUSTER_VULN_EDGE --references=storage.Cluster,cluster_cves:storage.ClusterCVE --read-only-store --search-scope CLUSTER_VULNERABILITIES,CLUSTER_VULN_EDGE,CLUSTERS
