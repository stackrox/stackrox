package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkPolicyApplicationUndoDeploymentRecord --table=networkpoliciesundodeployments --postgres-migration-seq 33 --migrate-from "rocksdb:networkpolicies-undodeployment"
