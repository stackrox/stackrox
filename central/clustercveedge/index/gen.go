package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object ClusterCVEEdge --singular ClusterCVEEdge --search-category CLUSTER_VULN_EDGE
//go:generate mockgen-wrapper Indexer
