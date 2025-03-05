package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Deployment --cached-store --search-category DEPLOYMENTS --references=storage.Image,namespaces:storage.NamespaceMetadata --search-scope IMAGE_VULNERABILITIES_V2,IMAGE_COMPONENTS_V2,IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS,PROCESS_INDICATORS,PODS
