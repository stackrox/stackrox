package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestG2GrandChild1 --table=test_g2_grand_child1 --search-category 66 --references test_g_grand_child1:storage.TestGGrandChild1,test_g3_grand_child1:storage.TestG3GrandChild1
