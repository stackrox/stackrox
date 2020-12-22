package deployment

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/processbaseline/evaluator"
	roleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	saStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once   sync.Once
	scorer Scorer
)

func initialize() {
	scorer = NewDeploymentScorer(alertDataStore.Singleton(), roleStore.Singleton(), bindingStore.Singleton(), saStore.Singleton(), evaluator.Singleton())
}

// GetScorer returns the singleton Scorer object to use when scoring risk.
func GetScorer() Scorer {
	once.Do(initialize)
	return scorer
}
