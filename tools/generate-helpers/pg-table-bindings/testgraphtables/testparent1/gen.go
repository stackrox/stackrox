package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent1 --table=test_parent1 --search-category 62 --references test_grandparents:storage.TestGrandparent,test_child1:storage.TestChild1
