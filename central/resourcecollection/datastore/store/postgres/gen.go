package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ResourceCollection --references=collections:storage.ResourceCollection --table=collections --search-category COLLECTIONS --cycle=EmbeddedCollections
