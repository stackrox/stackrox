package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NetworkBaseline --bucket=networkbaseline --key-func GetDeploymentId() --migrate-seq 27 --migrate-to network_baselines
