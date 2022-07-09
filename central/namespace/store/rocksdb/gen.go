package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NamespaceMetadata --bucket=namespaces --migrate-seq 2 --migrate-to namespaces
