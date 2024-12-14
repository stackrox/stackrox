package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageCVEV2 --table=image_cves_v2 --references=storage.Image --search-category IMAGE_VULNERABILITIES_V2 --search-scope IMAGE_VULNERABILITIES_V2,COMPONENT_VULN_EDGE_V2,IMAGE_COMPONENTS_V2,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS
