package rocksdb

//go:generate rocksdb-bindings-wrapper --type=PolicyCategory --bucket=policy_categories --uniq-key-func GetName()
