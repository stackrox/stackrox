package postgres

//go:generate pg-table-bindings-wrapper --type=storage.TestNoSerializedObj --no-serialized --search-category TEST_NO_SERIALIZED --schema-directory=pkg/postgres/schema --generate-data-model-helpers
