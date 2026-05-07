package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessIndicatorNoSerializedChild --no-serialized --search-category 88 --schema-directory=pkg/postgres/schema --table=process_indicator_no_serialized_child --repeated-field-strategy=signal.lineage_info:child_table --references storage.Deployment --generate-data-model-helpers
