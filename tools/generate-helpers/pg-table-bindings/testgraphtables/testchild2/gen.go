package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestChild2 --table=test_child2 --search-category 70 --references test_parent2:storage.TestParent2,test_grandparents:storage.TestGrandparent
