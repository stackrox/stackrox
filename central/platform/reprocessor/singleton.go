package reprocessor

import (
	alertDS "github.com/stackrox/rox/central/alert/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance PlatformReprocessor
)

func initialize() {
	soleInstance = New(alertDS.Singleton(), configDS.Singleton(), deploymentDS.Singleton(), platformmatcher.Singleton())
	soleInstance.Start()
}

// Singleton returns the sole instance of PlatformReprocessor.
func Singleton() PlatformReprocessor {
	once.Do(initialize)
	return soleInstance
}
