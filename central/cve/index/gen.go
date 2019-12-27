package index

//go:generate blevebindings-wrapper --write-options=false --options-path mappings --object CVE --singular CVE --search-category VULNERABILITIES
//go:generate mockgen-wrapper Indexer
