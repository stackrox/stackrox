package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestSingleKeyStruct --table=singlekey --search-category SEARCH_UNSET --options-path tools/generate-helpers/pg-table-bindings/test/options
