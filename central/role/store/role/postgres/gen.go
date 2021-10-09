package postgres

//go:generate pg-bindings-wrapper --type=Role --table=roles  --key-func GetName()
