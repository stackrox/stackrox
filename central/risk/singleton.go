package risk

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processWhitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
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
	scorer = NewScorer(alertDataStore.Singleton(), processIndicatorDataStore.Singleton(), processWhitelistDataStore.Singleton(), roleStore.Singleton(), bindingStore.Singleton(), saStore.Singleton())
}

// GetScorer returns the singleton Scorer object to use when scoring risk.
func GetScorer() Scorer {
	once.Do(initialize)
	return scorer
}
