package postgres

//go:generate pg-bindings-wrapper --type=NetworkPolicyApplicationUndoDeploymentRecord --table=networkpolicies-undodeployment =true --key-func GetDeploymentId()
