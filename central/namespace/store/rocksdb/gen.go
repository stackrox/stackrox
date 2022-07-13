package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NamespaceMetadata --bucket=namespaces --migration-seq 2 --migrate-to namespaces
