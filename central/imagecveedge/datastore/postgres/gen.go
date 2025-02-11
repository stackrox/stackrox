package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageCVEEdge --search-category IMAGE_VULN_EDGE --references=storage.Image,image_cves:storage.ImageCVE --read-only-store --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
