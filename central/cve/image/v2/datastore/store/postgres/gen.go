package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NormalizedCVE --table=cves --search-category NORMALIZED_CVES --search-scope NORMALIZED_CVES
