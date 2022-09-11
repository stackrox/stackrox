package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Deployment --search-category DEPLOYMENTS --references=storage.Image,namespaces:storage.NamespaceMetadata --search-scope IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,DEPLOYMENTS,NAMESPACES,CLUSTERS,PROCESS_INDICATORS --migration-seq 3 --migrate-from dackbox
