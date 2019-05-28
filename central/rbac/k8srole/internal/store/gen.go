package store

//go:generate boltbindings-wrapper --object=K8SRole --singular Role --get-return-exists --bucket=k8sroles --methods=get,upsert,delete,list
//go:generate mockgen-wrapper Store
