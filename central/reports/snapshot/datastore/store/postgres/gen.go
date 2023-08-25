package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ReportSnapshot --search-category REPORT_SNAPSHOT --references=storage.ReportConfiguration
