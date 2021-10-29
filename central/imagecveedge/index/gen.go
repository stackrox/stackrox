package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object ImageCVEEdge --singular ImageCVEEdge --search-category IMAGE_VULN_EDGE
//go:generate mockgen-wrapper Indexer
