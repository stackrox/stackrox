package index

//go:generate blevebindings-wrapper --object ProcessBaseline --singular Baseline --search-category PROCESS_BASELINES --generate-mock-indexer
//go:generate mockgen-wrapper Indexer
