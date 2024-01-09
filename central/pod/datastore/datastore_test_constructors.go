package datastore

import (
	"testing"

	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	piFilter "github.com/stackrox/rox/central/processindicator/filter"
	plopDS "github.com/stackrox/rox/central/processlisteningonport/datastore"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	processIndicatorStore, err := piDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	plopStore := plopDS.GetTestPostgresDataStore(t, pool)
	processIndicatorFilter := piFilter.Singleton()
	return NewPostgresDB(pool, processIndicatorStore, plopStore, processIndicatorFilter)
}
