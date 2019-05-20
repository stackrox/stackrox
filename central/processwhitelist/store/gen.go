package store

//go:generate boltbindings-wrapper --methods add,list,delete,get,update --bucket processWhitelists --object ProcessWhitelist --singular Whitelist
//go:generate mockgen-wrapper Store
