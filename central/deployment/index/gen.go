package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object Deployment --singular Deployment --search-category DEPLOYMENTS
//go:generate mockgen-wrapper Indexer
