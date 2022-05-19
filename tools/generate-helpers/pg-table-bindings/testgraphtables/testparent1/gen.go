package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent1 --search-category 62 --references storage.TestGrandparent,storage.TestChild1
