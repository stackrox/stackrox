package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestGrandChild1 --table=testgrandchild1 --search-category 64 --references testchild1:storage.TestChild1,testggrandchild1:storage.TestGGrandChild1
