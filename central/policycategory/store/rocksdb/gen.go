package rocksdb

//go:generate rocksdb-bindings-wrapper --type=PolicyCategory --bucket=policy_categories --uniq-key-func GetName() --migrate-seq 55 --migrate-to policy_categories
