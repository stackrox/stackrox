package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessIndicatorNoSerializedArrays --no-serialized --search-category 87 --schema-directory=pkg/postgres/schema --repeated-field-strategy=signal.lineage_info:array --references storage.Deployment
