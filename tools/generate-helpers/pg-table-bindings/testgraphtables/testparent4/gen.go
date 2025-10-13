package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent4 --search-category 113 --references storage.TestGrandparent --search-scope 109,103
