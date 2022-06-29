package rocksdb

//go:generate rocksdb-bindings-wrapper --type=InitBundleMeta --bucket=clusterinitbundles --cache --migrate-seq 7 --migrate-to cluster_init_bundles
