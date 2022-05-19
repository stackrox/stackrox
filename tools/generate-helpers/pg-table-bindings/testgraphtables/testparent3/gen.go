package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent3 --table=test_parent3 --search-category 69 --references test_grandparents:storage.TestGrandparent
