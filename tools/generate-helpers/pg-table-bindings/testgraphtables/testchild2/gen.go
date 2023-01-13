package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestChild2 --search-category 104 --references storage.TestParent2,storage.TestGrandparent
