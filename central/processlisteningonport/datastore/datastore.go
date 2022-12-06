package datastore

import (
	"context"
	"fmt"

	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
)

// GetOptions specifies how to get information from the database, simply
// filtering by the namespace, or both namespace and deployment
type GetOptions struct {
	DeploymentID *string
	Namespace    *string
}

func (opts *GetOptions) String() string {
	return fmt.Sprintf("GetOptions{Namespace: %s, DeploymentID: %s}",
		*opts.Namespace, *opts.DeploymentID)
}

// DataStore interface for ProcessListeningOnPort object interaction with the database
//go:generate mockgen-wrapper
type DataStore interface {
	AddProcessListeningOnPort(context.Context, ...*storage.ProcessListeningOnPort) error
	GetProcessListeningOnPort(
		ctx context.Context,
		opts GetOptions,
	) (map[string][]*storage.ProcessListeningOnPort, error)
}

// New creates a data store object to access the database. Since some
// operations require join with ProcessIndicator table, both PLOP store and
// ProcessIndicator datastore are needed.
func New(
	plopStorage postgres.Store,
	indicatorDataStore processIndicatorStore.DataStore,
) DataStore {
	ds := newDatastoreImpl(plopStorage, indicatorDataStore)
	return ds
}
