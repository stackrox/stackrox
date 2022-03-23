package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Pod --table=pods --search-category PODS --options-path central/pod/mappings
