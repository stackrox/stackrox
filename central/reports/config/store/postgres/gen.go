package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ReportConfiguration --search-category REPORT_CONFIGURATIONS --search-scope REPORT_SNAPSHOT --references=storage.Notifier
