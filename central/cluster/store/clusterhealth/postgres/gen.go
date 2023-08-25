package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ClusterHealthStatus --references=storage.Cluster --search-category=CLUSTER_HEALTH
