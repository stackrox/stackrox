package postgres

//go:generate pg-bindings-wrapper --type=NetworkBaseline --table=networkbaseline --key-func GetDeploymentId()
