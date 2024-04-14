package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/processlisteningonport/store"
	store "github.com/stackrox/rox/central/runtimeconfiguration/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore interface for ProcessListeningOnPort object interaction with the database
//
//go:generate mockgen-wrapper
type DataStore interface {
}

// New creates a data store object to access the database. Since some
// operations require join with ProcessIndicator table, both PLOP store and
// ProcessIndicator datastore are needed.
func New(
	plopStorage store.Store,
	pool postgres.DB,
) DataStore {
	ds := newDatastoreImpl(store, pool)
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
	rcStore := store.NewFullStore(pool)
	if err != nil {
		log.Infof("getting test store %v", err)
	}
	return newDatastoreImpl(rcStore, indicatorDS, pool)
}
