package postgres

//go:generate pg-table-bindings-wrapper --type=NetworkPolicyApplicationUndoDeploymentRecord --table=networkpolicies-undodeployment =true --key-func GetDeploymentId()
