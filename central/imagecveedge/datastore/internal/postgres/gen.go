package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageCVEEdge --search-category IMAGE_VULN_EDGE --references=storage.Image,image_cves:storage.CVE --join-table true
