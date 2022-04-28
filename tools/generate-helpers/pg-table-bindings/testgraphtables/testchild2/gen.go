package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestChild2 --table=testchild2 --search-category 70 --references testparent2:storage.TestParent2,testgrandparent:storage.TestGrandparent
