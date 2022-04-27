package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageCVEEdge --table=image_cve_relations --search-category IMAGE_VULN_EDGE --references=images:storage.Image,image_cves:storage.CVE --join-table true
