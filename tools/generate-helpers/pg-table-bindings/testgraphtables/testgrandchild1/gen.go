package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestGrandChild1 --table=test_grand_child1 --search-category 64 --references test_child1:storage.TestChild1,test_g_grand_child1:storage.TestGGrandChild1
