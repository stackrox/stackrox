package postgres

//go:generate pgsearchbindings-wrapper --table clusters --type Cluster --options-path "central/cluster/index/mappings" --singular Cluster --search-category CLUSTERS
