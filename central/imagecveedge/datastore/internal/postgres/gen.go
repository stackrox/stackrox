package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageCVEEdge --table=image_cve_relations --search-category IMAGE_VULN_EDGE --options-path central/imagecveedge/mappings  --references=images:storage.Image --join-table true
