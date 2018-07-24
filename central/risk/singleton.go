package risk

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	"bitbucket.org/stack-rox/apollo/central/dnrintegration/datastore"
)

var (
	once   sync.Once
	scorer Scorer
)

func initialize() {
	scorer = NewScorer(alertDataStore.Singleton(), datastore.Singleton())
}

// GetScorer returns the singleton Scorer object to use when scoring risk.
func GetScorer() Scorer {
	once.Do(initialize)
	return scorer
}
