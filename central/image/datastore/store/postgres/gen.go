package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Image --search-category IMAGES --schema-only --postgres-migration-seq 4 --migrate-from "dackbox:images"
