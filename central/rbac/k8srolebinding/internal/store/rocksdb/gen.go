package rocksdb

//go:generate rocksdb-bindings-wrapper --type=K8SRoleBinding --bucket=rolebindings --migrate-seq 45 --migrate-to role_bindings
