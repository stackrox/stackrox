package postgres

//go:generate pg-table-bindings-wrapper --type=SimpleAccessScope --table=simple_access_scopes  --uniq-key-func GetName()
