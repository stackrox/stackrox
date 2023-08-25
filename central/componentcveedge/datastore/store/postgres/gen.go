package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdge --table=image_component_cve_edges --search-category COMPONENT_VULN_EDGE --references=storage.ImageComponent,image_cves:storage.ImageCVE --read-only-store --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
