package risk

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once   sync.Once
	scorer Scorer
)

func initialize() {
	scorer = NewScorer(alertDataStore.Singleton())
}

// GetScorer returns the singleton Scorer object to use when scoring risk.
func GetScorer() Scorer {
	once.Do(initialize)
	return scorer
}
