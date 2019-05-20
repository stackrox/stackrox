package store

//!go:generate boltbindings-wrapper --object=Certificate --singular=Certificate --bucket=clientCAs --id-func=GetId --methods=list,upsert_many,delete
//go:generate mockgen-wrapper Store
