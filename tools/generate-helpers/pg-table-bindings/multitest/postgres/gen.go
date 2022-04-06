package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestMultiKeyStruct --table=multikey --search-category SEARCH_UNSET --options-path tools/generate-helpers/pg-table-bindings/multitest/options
