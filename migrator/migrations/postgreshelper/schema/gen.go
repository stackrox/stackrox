package schema

//go:generate pg-schema-migration-helper --type=storage.TestSingleKeyStruct --search-category SEARCH_UNSET --get-all-func
//go:generate pg-schema-migration-helper --type=storage.TestSingleUUIDKeyStruct --search-category SEARCH_UNSET --get-all-func
//go:generate pg-schema-migration-helper --type=storage.TestStruct --search-category SEARCH_UNSET
//go:generate pg-schema-migration-helper --type=storage.TestParent4 --search-category 72 --references storage.TestGrandparent --search-scope 61,74
//go:generate pg-schema-migration-helper --type=storage.TestParent3 --search-category 69 --references storage.TestGrandparent
//go:generate pg-schema-migration-helper --type=storage.TestChild1P4 --search-category 74 --references storage.TestParent4 --search-scope 74
//go:generate pg-schema-migration-helper --type=storage.TestGGrandChild1 --search-category 65
//go:generate pg-schema-migration-helper --type=storage.TestParent2 --search-category 68 --references storage.TestGrandparent
//go:generate pg-schema-migration-helper --type=storage.TestG2GrandChild1 --search-category 66 --references storage.TestGGrandChild1,storage.TestG3GrandChild1
//go:generate pg-schema-migration-helper --type=storage.TestChild2 --search-category 70 --references storage.TestParent2,storage.TestGrandparent
//go:generate pg-schema-migration-helper --type=storage.TestParent1 --search-category 62 --references storage.TestGrandparent,storage.TestChild1
//go:generate pg-schema-migration-helper --type=storage.TestGrandparent --search-category 61
//go:generate pg-schema-migration-helper --type=storage.TestChild1 --search-category 63
//go:generate pg-schema-migration-helper --type=storage.TestGrandChild1 --search-category 64 --references storage.TestChild1,storage.TestGGrandChild1
//go:generate pg-schema-migration-helper --type=storage.TestG3GrandChild1 --search-category 67
//go:generate pg-schema-migration-helper --type=storage.TestShortCircuit --search-category 71 --references storage.TestChild1,storage.TestG2GrandChild1
