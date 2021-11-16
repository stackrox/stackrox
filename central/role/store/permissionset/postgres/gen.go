package postgres

//go:generate pg-table-bindings-wrapper --type=PermissionSet --table=permission_sets  --uniq-key-func GetName()
