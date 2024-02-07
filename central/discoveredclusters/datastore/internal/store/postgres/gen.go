package postgres

//go:generate pg-table-bindings-wrapper --type=storage.DiscoveredCluster --search-category DISCOVERED_CLUSTERS --references=storage.CloudSource
