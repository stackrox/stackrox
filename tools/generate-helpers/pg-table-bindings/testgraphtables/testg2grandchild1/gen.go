package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestG2GrandChild1 --table=testg2grandchild1 --search-category 66 --references testggrandchild1:storage.TestGGrandChild1,testg3granchild1:storage.TestG3GrandChild1
