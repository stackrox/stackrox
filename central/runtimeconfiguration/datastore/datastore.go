package datastore

import (
	"context"
	// "testing"
	// "time"

	store "github.com/stackrox/rox/central/runtimeconfiguration/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore interface for ProcessListeningOnPort object interaction with the database
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetRuntimeConfiguration(ctx context.Context) (*storage.RuntimeFilteringConfiguration, error)
	SetRuntimeConfiguration(ctx context.Context, runtimeConfiguration *storage.RuntimeFilteringConfiguration) error
}

// New creates a data store object to access the database. Since some
// operations require join with ProcessIndicator table, both PLOP store and
// ProcessIndicator datastore are needed.
func New(
	configStore store.Store,
	pool postgres.DB,
) DataStore {
	ds := newDatastoreImpl(configStore, pool)
	return ds
}

//// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
// func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
//	rcStore := store.NewFullStore(pool)
//	return newDatastoreImpl(rcStore, pool)
//}
