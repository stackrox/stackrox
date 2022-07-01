package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent4 --search-category 72 --references storage.TestGrandparent --search-scope 61,74
