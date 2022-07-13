package rocksdb

//go:generate rocksdb-bindings-wrapper --type=InitBundleMeta --bucket=clusterinitbundles --cache --migration-seq 11 --migrate-to cluster_init_bundles
