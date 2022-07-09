package postgres

//go:generate pg-table-bindings-wrapper --type=storage.InitBundleMeta --table=cluster_init_bundles --permission-checker permissionCheckerSingleton() --postgres-migration-seq 11 --migrate-from "rocksdb:clusterinitbundles"
