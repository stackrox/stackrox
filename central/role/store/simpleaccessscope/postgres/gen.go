package postgres

//go:generate pg-bindings-wrapper --type=SimpleAccessScope --table=simple_access_scopes  --uniq-key-func GetName()
