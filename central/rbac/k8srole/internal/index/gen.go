package index

//go:generate blevebindings-wrapper --tag k8s_role --write-options=false --object-path-name rbac/k8srole --options-path mappings --object K8SRole --singular K8SRole --search-category ROLES
//go:generate mockgen-wrapper Indexer
