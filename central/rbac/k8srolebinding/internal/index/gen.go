package index

//go:generate blevebindings-wrapper --tag k8s_role_binding --write-options=false --object-path-name rbac/k8srolebinding --options-path mappings --object K8SRoleBinding --singular K8SRoleBinding --search-category ROLEBINDINGS --generate-mock-indexer
