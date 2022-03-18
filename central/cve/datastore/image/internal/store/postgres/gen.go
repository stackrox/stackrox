package postgres

//go:generate pg-table-bindings-wrapper --type=storage.CVE --table=image_cves --search-category IMAGE_VULNERABILITIES --options-path central/cve/mappings
