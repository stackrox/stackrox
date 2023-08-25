package postgres

// Cluster store does not use Postgres CopyFrom operation to copy data into DB. This is because copyFrom requires an
// explicit delete prior to copy consequently prohibiting references to clusters table.
//go:generate pg-table-bindings-wrapper --type=storage.Cluster --search-category CLUSTERS --no-copy-from
