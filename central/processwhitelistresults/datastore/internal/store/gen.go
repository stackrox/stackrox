package store

//go:generate boltbindings-wrapper --object=ProcessWhitelistResults --singular=WhitelistResults --bucket=processWhitelistResults --id-func=GetDeploymentId --methods=get,upsert,delete
//go:generate mockgen-wrapper Store
