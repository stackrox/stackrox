package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageCVE --table=image_cves --search-category IMAGE_VULNERABILITIES --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
