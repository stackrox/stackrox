package postgres

//go:generate pg-table-bindings-wrapper --type=storage.CustomResource --table=custom_resources --for-sac --search-category CUSTOM_RESOURCES --search-scope CUSTOM_RESOURCES,NAMESPACES,CLUSTERS --default-sort search.CustomResourceName.String()
