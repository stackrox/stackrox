package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ClusterHealthStatus --references=storage.Cluster --search-category=CLUSTER_HEALTH
// To regenerate migration:
// --migration-seq 10 --migrate-from rocksdb
