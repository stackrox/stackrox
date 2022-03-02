package postgres

//Todo, do we need something like --options-path central/alert/mappings
//go:generate pg-table-bindings-wrapper --type=storage.ProcessIndicator --table=process_indicators --search-category PROCESS_INDICATORS --options-path central/processindicator/mappings
