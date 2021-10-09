package postgres

//go:generate pg-bindings-wrapper --type=Cluster --table=clusters  --uniq-key-func GetName()
