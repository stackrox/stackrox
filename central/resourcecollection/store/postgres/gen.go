package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ResourceCollection --table=collections --search-category COLLECTIONS COLLECTION_NAME
//--references=collections:storage.ResourceCollection
