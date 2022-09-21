package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ResourceCollection --table=collections --search-category COLLECTIONS --references=storage.ResourceCollection COLLECTION_NAME
