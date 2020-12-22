package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ProcessBaselineResults --bucket=processWhitelistResults --key-func=GetDeploymentId()
