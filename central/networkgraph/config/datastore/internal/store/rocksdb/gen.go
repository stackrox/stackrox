package rocksdb

//go:generate rocksdb-bindings-wrapper --type=NetworkGraphConfig --bucket=networkgraphconfig --migrate-seq 30 --migrate-to network_graph_configs
