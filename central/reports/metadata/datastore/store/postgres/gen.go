package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ReportMetadata --search-category REPORT_METADATA --references=storage.ReportConfiguration
