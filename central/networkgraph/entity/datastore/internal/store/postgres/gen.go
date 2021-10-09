package postgres

//go:generate pg-bindings-wrapper --type=NetworkEntity --table=networkentity  --key-func GetInfo().GetId()
