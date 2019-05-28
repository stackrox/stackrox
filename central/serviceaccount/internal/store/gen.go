package store

//go:generate boltbindings-wrapper --object=ServiceAccount --singular=ServiceAccount --get-return-exists --bucket=service_accounts --methods=get,get_many,upsert,delete,list
//go:generate mockgen-wrapper Store
