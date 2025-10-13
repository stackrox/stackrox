package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent1 --search-category 110 --references storage.TestGrandparent,storage.TestChild1
