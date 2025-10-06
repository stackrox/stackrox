package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Risk --search-category RISKS --default-sort search.RiskScore.String() --reverse-default-sort
