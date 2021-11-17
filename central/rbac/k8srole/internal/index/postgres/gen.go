package postgres

//go:generate pgsearchbindings-wrapper  --registered-type=K8sRole --write-options=false --options-path "central/rbac/k8srole/mappings" --type K8SRole --singular K8SRole --search-category ROLES
