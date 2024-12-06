package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdgeV2 --table=image_component_cve_edges_v2 --search-category COMPONENT_VULN_EDGE_V2 --references=storage.ImageComponentV2,image_cves_v2:storage.ImageCVEV2 --read-only-store --search-scope IMAGE_VULNERABILITIES_V2,COMPONENT_VULN_EDGE_V2,IMAGE_COMPONENTS_V2,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
