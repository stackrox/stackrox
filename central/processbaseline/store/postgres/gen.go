package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessBaseline --table=processbaselines --search-category PROCESS_BASELINES --options-path central/processbaseline/index/mappings
