package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent2 --table=testparent2 --search-category 68 --references testgrandparent:storage.TestGrandparent
