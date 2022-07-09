package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NamespaceMetadata --table=namespaces --search-category NAMESPACES --references=storage.Cluster --postgres-migration-seq 2 --migrate-from "rocksdb:namespaces"
