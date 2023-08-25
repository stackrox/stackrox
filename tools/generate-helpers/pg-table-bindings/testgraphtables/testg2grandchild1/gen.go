package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestG2GrandChild1 --search-category 105 --references storage.TestGGrandChild1,storage.TestG3GrandChild1
