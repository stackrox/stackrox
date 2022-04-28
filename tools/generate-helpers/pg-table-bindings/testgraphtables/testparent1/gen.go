package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestParent1 --table=testparent1 --search-category 62 --references testgrandparent:storage.TestGrandparent,testchild1:storage.TestChild1
