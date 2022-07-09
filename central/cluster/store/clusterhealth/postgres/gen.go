package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ClusterHealthStatus --references=storage.Cluster --search-category=CLUSTER_HEALTH --postgres-migration-seq 10 --migrate-from "rocksdb:clusters_health_status"
