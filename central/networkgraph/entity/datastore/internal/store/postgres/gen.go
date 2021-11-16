package postgres

//go:generate pg-table-bindings-wrapper --type=NetworkEntity --table=networkentity  --key-func GetInfo().GetId()
