package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Image --table=images --search-category IMAGES --options-path central/image/mappings
