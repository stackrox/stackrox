package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ClusterHealthStatus --table=cluster_health_status --search-category CLUSTER_HEALTH_STATUS --references=clusters:storage.Cluster
