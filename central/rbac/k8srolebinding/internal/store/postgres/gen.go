package postgres

//go:generate pg-table-bindings-wrapper --registered-type=storage.K8sRoleBinding --type=storage.K8SRoleBinding --table=k8s_role_bindings --search-category ROLEBINDINGS
