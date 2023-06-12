package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ReportMetadata --table=report_metadatas --search-category REPORT_METADATA --references=storage.ReportConfiguration
