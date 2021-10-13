package postgres

//go:generate pgsearchbindings-wrapper --table rolebindings --write-options=false --options-path "central/rbac/k8srolebinding/mappings" --type K8SRoleBinding --singular K8sRoleBinding --search-category ROLEBINDINGS
