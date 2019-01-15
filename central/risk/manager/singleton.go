package manager

import (
	"sync"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	multiplierDS "github.com/stackrox/rox/central/multiplier/store"
	"github.com/stackrox/rox/central/risk"
)

var (
	once    sync.Once
	manager Manager
)

func initialize() {
	var err error
	manager, err = New(deploymentDS.Singleton(), multiplierDS.Singleton(), risk.GetScorer())
	if err != nil {
		panic(err)
	}
}

// Singleton provides the singleton Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return manager
}
