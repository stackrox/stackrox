package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent2 --search-category 111 --references storage.TestGrandparent
