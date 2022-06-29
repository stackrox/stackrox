package postgres

//go:generate pg-table-bindings-wrapper --type=storage.SensorUpgradeConfig --singleton --postgres-migration-seq 51 --migrate-from "boltdb:sensor-upgrade-config"
