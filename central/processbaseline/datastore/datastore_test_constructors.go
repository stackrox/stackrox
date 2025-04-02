package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/processbaseline/search"
	pgStore "github.com/stackrox/rox/central/processbaseline/store/postgres"
	"github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	searcher := search.New(store)
	resultsStore := datastore.GetTestPostgresDataStore(t, pool)
	indicatorStore := processIndicatorDatastore.GetTestPostgresDataStore(t, pool)
	return New(store, searcher, resultsStore, indicatorStore)
}
