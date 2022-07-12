package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NetworkBaseline --bucket=networkbaseline --key-func GetDeploymentId() --migration-seq 28 --migrate-to network_baselines
