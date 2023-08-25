package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestGrandChild1 --search-category 108 --references storage.TestChild1,storage.TestGGrandChild1
