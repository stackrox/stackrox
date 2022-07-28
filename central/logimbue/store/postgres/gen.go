package postgres

//go:generate pg-table-bindings-wrapper --type=storage.LogImbue --get-all-func
// To regenerate migration:
// //go:generate pg-table-bindings-wrapper --type=storage.LogImbue --get-all-func --migration-seq 27 --migrate-from boltdb
