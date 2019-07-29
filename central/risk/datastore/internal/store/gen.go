package store

//go:generate boltbindings-wrapper --methods get,list,get_many,upsert,delete --bucket risk --object Risk --singular Risk
//go:generate mockgen-wrapper Store
