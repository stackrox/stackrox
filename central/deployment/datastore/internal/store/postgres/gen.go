package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Deployment --cached-store --search-category DEPLOYMENTS --references=storage.Image,namespaces:storage.NamespaceMetadata,imagesV2:storage.ImageV2 --search-scope IMAGE_VULNERABILITIES_V2,IMAGE_COMPONENTS_V2,IMAGE_VULNERABILITIES,COMPONENT_VULN_EDGE,IMAGE_COMPONENTS,IMAGE_COMPONENT_EDGE,IMAGE_VULN_EDGE,IMAGES,IMAGES_V2,DEPLOYMENTS,NAMESPACES,CLUSTERS,PROCESS_INDICATORS,PODS --default-sort search.DeploymentPriority.String() --transform-sort-options DeploymentsSchema.OptionsMap
