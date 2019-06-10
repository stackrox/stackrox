package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object ProcessIndicator --singular ProcessIndicator --search-category PROCESS_INDICATORS
//go:generate mockgen-wrapper Indexer
