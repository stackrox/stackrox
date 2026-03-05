package datastore

import (
	"testing"

	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) imageDatastore.DataStore {
	v1Store := imageDatastore.GetTestPostgresDataStore(t, pool)
	v2Store := imageV2Datastore.GetTestPostgresDataStore(t, pool)
	return New(v1Store, v2Store)
}
