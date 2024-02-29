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
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	searcher, err := search.New(store)
	if err != nil {
		return nil, err
	}
	resultsStore, err := datastore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	indicatorStore, err := processIndicatorDatastore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	return New(store, searcher, resultsStore, indicatorStore), nil
}
