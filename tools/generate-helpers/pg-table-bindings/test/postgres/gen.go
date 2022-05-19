package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestSingleKeyStruct --table=test_single_key_structs --search-category SEARCH_UNSET --get-all-func
