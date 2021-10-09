package postgres

//go:generate pg-bindings-wrapper --type=ProcessBaselineResults --table=processWhitelistResults --key-func=GetDeploymentId()
