package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object ImageComponent --singular ImageComponent --search-category IMAGE_COMPONENTS
//go:generate mockgen-wrapper Indexer
