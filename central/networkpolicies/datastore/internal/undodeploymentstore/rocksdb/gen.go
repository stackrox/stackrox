package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NetworkPolicyApplicationUndoDeploymentRecord --bucket=networkpolicies-undodeployment --cache=true --key-func GetDeploymentId()
