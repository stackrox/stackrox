package updater

import (
	activeComponent "github.com/stackrox/rox/central/activecomponent/datastore"
	"github.com/stackrox/rox/central/activecomponent/updater/aggregator"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageStore "github.com/stackrox/rox/central/image/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	acUpdater Updater
)

func initialize() {
	acUpdater = New(
		activeComponent.Singleton(),
		deploymentDataStore.Singleton(),
		processIndicatorDataStore.Singleton(),
		imageStore.Singleton(),
		aggregator.Singleton(),
	)
}

// Singleton provides the active component updater instance
func Singleton() Updater {
	once.Do(initialize)
	return acUpdater
}
