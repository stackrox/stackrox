package datastore

import (
	"context"

	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
)

//go:generate mockgen-wrapper
type DataStore interface {
	AddProcessListeningOnPort(context.Context, ...*storage.ProcessListeningOnPort) error
	GetProcessListeningOnPortForDeployment(
		ctx context.Context,
		deploymentId string,
	) (*storage.ProcessListeningOnPort, error)
}

func New(
	plopStorage postgres.Store,
	indicatorDataStore processIndicatorStore.DataStore,
) DataStore {
	ds := newDatastoreImpl(plopStorage, indicatorDataStore)
	return ds
}
