package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessIndicatorNoSerialized --no-serialized --search-category 85 --schema-directory=pkg/postgres/schema --generate-data-model-helpers
