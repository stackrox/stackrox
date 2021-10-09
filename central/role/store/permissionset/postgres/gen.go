package postgres

//go:generate pg-bindings-wrapper --type=PermissionSet --table=permission_sets  --uniq-key-func GetName()
