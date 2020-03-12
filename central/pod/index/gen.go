package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object Pod --singular Pod --search-category PODS
//go:generate mockgen-wrapper Indexer
