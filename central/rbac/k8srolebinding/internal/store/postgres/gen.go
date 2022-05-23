package postgres

//go:generate pg-table-bindings-wrapper --registered-type=storage.K8sRoleBinding --type=storage.K8SRoleBinding --table=role_bindings --search-category ROLEBINDINGS
