package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NetworkBaseline --bucket=networkbaseline --cache=true --key-func GetDeploymentId()
