package postgres

//go:generate pg-table-bindings-wrapper --type=storage.LogImbue --get-all-func
// To regenerate migration, add:
// --migration-seq 27 --migrate-from boltdb
