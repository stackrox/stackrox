package postgres

//go:generate pg-table-bindings-wrapper --type=storage.CVE --table=node_cves --search-category NODE_VULNERABILITIES --options-path central/cve/mappings
