package datastore

import (
	"context"

	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

//go:generate mockgen-wrapper
type DataStore interface {
	AddProcessListeningOnPort(context.Context, ...*storage.ProcessListeningOnPort) error

	GetProcessListeningOnPortForDeployment(
	ctx context.Context,
	namespace string,
	deploymentId string,
	) ([]*storage.ProcessListeningOnPort, error)

	GetProcessListeningOnPortForNamespace(
	ctx context.Context,
	namespace string,
	) ([]*v1.ProcessListeningOnPortWithDeploymentId, error)
}

func New(
	plopStorage postgres.Store,
	indicatorDataStore processIndicatorStore.DataStore,
) DataStore {
	ds := newDatastoreImpl(plopStorage, indicatorDataStore)
	return ds
}
