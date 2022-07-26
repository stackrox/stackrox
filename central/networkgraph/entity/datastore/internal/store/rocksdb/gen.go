package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NetworkEntity --bucket=networkentity --cache --key-func GetInfo().GetId() --migration-seq 29 --migrate-to network_entities
