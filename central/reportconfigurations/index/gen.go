package index

//go:generate blevebindings-wrapper --write-options=false --options-path ../reportconfigurations/mappings --object ReportConfiguration --singular ReportConfiguration --search-category REPORT_CONFIGURATIONS
//go:generate mockgen-wrapper Indexer
