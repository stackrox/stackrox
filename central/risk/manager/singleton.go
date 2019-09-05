package manager

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/scorer"
	serviceAccDS "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager Manager
)

func initialize() {
	var err error
	manager, err = New(
		deploymentDS.Singleton(),
		serviceAccDS.Singleton(),
		riskDS.Singleton(),
		scorer.GetScorer())
	if err != nil {
		panic(err)
	}
}

// Singleton provides the singleton Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return manager
}
