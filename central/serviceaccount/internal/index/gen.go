package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object ServiceAccount --singular ServiceAccount --search-category SERVICE_ACCOUNTS
//go:generate mockgen-wrapper Indexer
