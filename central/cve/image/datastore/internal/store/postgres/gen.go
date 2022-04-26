package postgres

//go:generate pg-table-bindings-wrapper --type=storage.CVE --table=image_cves --search-category VULNERABILITIES --options-path central/cve/mappings
