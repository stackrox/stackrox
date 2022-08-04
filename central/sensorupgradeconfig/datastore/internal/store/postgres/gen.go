package postgres

//go:generate pg-table-bindings-wrapper --type=storage.SensorUpgradeConfig --singleton --migration-seq 48 --migrate-from boltdb
