package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Policy --table=policy --search-category POLICIES --options-path central/policy/index/mappings
