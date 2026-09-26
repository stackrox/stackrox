package postgres

//go:generate pg-table-bindings-wrapper --type=storage.AIMetadata --table=ai_metadata --for-sac --search-category AI_METADATA --references=custom_resources:storage.CustomResource --search-scope AI_METADATA,CUSTOM_RESOURCES,NAMESPACES,CLUSTERS --default-sort search.AIModelName.String()
