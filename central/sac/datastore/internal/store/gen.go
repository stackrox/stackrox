package store

//go:generate boltbindings-wrapper --methods upsert,get,list,delete,upsert_many --bucket authzPlugins --object AuthzPluginConfig --singular AuthzPluginConfig
//go:generate mockgen-wrapper Store
