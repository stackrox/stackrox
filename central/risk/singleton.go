package risk

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/dnrintegration/datastore"
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
