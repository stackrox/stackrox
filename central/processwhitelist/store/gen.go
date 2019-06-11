package store

//go:generate boltbindings-wrapper --methods add,list,delete,get,update,get_many --bucket processWhitelists2 --object ProcessWhitelist --singular Whitelist
//go:generate mockgen-wrapper Store
