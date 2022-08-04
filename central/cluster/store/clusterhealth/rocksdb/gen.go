package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ClusterHealthStatus --bucket=clusters_health_status --cache --uniq-key-func GetId() --migration-seq 10 --migrate-to cluster_health_statuses
