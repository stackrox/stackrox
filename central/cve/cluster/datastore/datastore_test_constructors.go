package datastore

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackrox/rox/central/cve/cluster/datastore/search"
	"github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.NewFullStore(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher)
}
