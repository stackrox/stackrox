package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object ComponentCVEEdge --singular ComponentCVEEdge --search-category COMPONENT_VULN_EDGE
//go:generate mockgen-wrapper Indexer
