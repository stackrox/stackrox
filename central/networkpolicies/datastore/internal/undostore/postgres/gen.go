package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NetworkPolicyApplicationUndoRecord --table=networkpolicyapplicationundorecords --migration-seq 34 --migrate-from boltdb
