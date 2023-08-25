package postgres

//go:generate pg-table-bindings-wrapper --registered-type=storage.K8sRole --type=storage.K8SRole --table=k8s_roles --search-category ROLES
