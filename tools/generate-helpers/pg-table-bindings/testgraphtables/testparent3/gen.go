package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent3 --table=testparent3 --search-category 69 --references testgrandparent:storage.TestGrandparent
