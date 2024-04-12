package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ResourceCollection --references=collections:storage.ResourceCollection --table=runtime_configuration_collections --search-category COLLECTIONS --cycle=EmbeddedCollections
