package index

//go:generate blevebindings-wrapper --tag k8s_role_binding --write-options=false --object-path-name rbac/k8srolebinding --options-path mappings --object K8SRoleBinding --singular K8sRoleBinding --search-category ROLEBINDINGS
//go:generate mockgen-wrapper Indexer
