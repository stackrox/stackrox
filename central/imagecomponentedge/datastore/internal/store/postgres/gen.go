package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --search-category IMAGE_COMPONENT_EDGE --references=storage.Image,storage.ImageComponent --read-only-store --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
