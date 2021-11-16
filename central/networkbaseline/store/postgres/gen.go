package postgres

//go:generate pg-table-bindings-wrapper --type=NetworkBaseline --table=networkbaseline --key-func GetDeploymentId()
