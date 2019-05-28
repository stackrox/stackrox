package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object Secret --singular Secret --search-category SECRETS
//go:generate mockgen-wrapper Indexer
