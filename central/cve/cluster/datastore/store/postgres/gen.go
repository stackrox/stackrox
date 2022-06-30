package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ClusterCVE --table=cluster_cves --search-category CLUSTER_VULNERABILITIES
