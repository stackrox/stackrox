package updater

import (
	"context"

	activeComponent "github.com/stackrox/rox/central/activecomponent/datastore"
	"github.com/stackrox/rox/central/activecomponent/updater/aggregator"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	imageStore "github.com/stackrox/rox/central/image/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/simplecache"
)

//go:generate mockgen-wrapper

// Updater helps create active components
type Updater interface {
	PopulateExecutableCache(ctx context.Context, image *storage.Image) error
	Update()
}

// New returns a new instance of ActiveComponent Updater.
func New(acStore activeComponent.DataStore, deploymentStore deploymentStore.DataStore, piStore processIndicatorStore.DataStore, imageStore imageStore.DataStore, aggregator aggregator.ProcessAggregator) Updater {
	return &updaterImpl{
		acStore:         acStore,
		deploymentStore: deploymentStore,
		piStore:         piStore,
		imageStore:      imageStore,
		aggregator:      aggregator,

		executableCache: simplecache.New(),
	}
}
