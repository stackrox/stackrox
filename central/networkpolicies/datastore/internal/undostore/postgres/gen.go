package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkPolicyApplicationUndoRecord --table=networkpolicyapplicationundorecords --postgres-migration-seq 33 --migrate-from "boltdb:networkpolicies-undo"
