package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object Image --singular Image --search-category IMAGES
//go:generate mockgen-wrapper Indexer
