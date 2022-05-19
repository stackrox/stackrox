package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent2 --table=test_parent2 --search-category 68 --references test_grandparents:storage.TestGrandparent
