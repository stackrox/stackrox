package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkPolicyApplicationUndoDeploymentRecord --table=networkpoliciesundodeployments --migration-seq 33 --migrate-from rocksdb
