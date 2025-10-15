package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --references=storage.Image,storage.ImageComponent --read-only-store --schema-only
