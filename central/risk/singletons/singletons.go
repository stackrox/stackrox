package singletons

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	"bitbucket.org/stack-rox/apollo/central/risk"
)

var (
	once   sync.Once
	scorer *risk.Scorer
)

func initialize() {
	scorer = risk.NewScorer(alertDataStore.Singleton())
}

// GetScorer returns the singleton Scorer object to use when scoring risk.
func GetScorer() *risk.Scorer {
	once.Do(initialize)
	return scorer
}
