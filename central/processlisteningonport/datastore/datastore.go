package datastore

import (
	"context"

	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store"
	"github.com/stackrox/rox/generated/storage"
)

// WalkFn is a convenient type alias to use for the Walk function
type WalkFn = func(plop *storage.ProcessListeningOnPortStorage) error

// DataStore interface for ProcessListeningOnPort object interaction with the database
//
//go:generate mockgen-wrapper
type DataStore interface {
	AddProcessListeningOnPort(context.Context, ...*storage.ProcessListeningOnPortFromSensor) error
	GetProcessListeningOnPort(
		ctx context.Context,
		deployment string,
	) ([]*storage.ProcessListeningOnPort, error)
	WalkAll(ctx context.Context, fn WalkFn) error
	RemoveProcessListeningOnPort(ctx context.Context, ids []string) error
}

// New creates a data store object to access the database. Since some
// operations require join with ProcessIndicator table, both PLOP store and
// ProcessIndicator datastore are needed.
func New(
	plopStorage store.Store,
	indicatorDataStore processIndicatorStore.DataStore,
) DataStore {
	ds := newDatastoreImpl(plopStorage, indicatorDataStore)
	return ds
}
