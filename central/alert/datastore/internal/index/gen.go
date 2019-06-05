package index

//go:generate blevebindings-wrapper --object-path-name alert --write-options=false --options-path mappings --object ListAlert --singular ListAlert --search-category ALERTS
//go:generate mockgen-wrapper Indexer
