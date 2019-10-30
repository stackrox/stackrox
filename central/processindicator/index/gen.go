package index

//go:generate blevebindings-wrapper --tx-manager --options-path mappings --object ProcessIndicator --singular ProcessIndicator --search-category PROCESS_INDICATORS
//go:generate mockgen-wrapper Indexer
