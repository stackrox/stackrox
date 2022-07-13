package rocksdb

//go:generate rocksdb-bindings-wrapper --type=K8SRole --bucket=k8sroles --migration-seq 26 --migrate-to k8s_roles
