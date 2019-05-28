package store

//go:generate boltbindings-wrapper --object=K8SRoleBinding --singular RoleBinding --get-return-exists --bucket=rolebindings --methods=get,get_many,upsert,delete,list
//go:generate mockgen-wrapper Store
