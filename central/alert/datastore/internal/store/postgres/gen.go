package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Alert --table=alerts --search-category ALERTS --options-path central/alert/mappings
