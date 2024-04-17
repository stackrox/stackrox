package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ResourceCollection --references=collections:storage.ResourceCollection --table=runtime_configuration_collections --search-category RUNTIME_CONFIGURATION_COLLECTIONS --cycle=EmbeddedCollections
