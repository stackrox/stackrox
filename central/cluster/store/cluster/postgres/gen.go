package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Cluster --table=clusters --search-category CLUSTERS --options-path central/cluster/index/mappings
