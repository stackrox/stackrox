package evaluator

import (
	baselinesStore "github.com/stackrox/rox/central/processbaseline/datastore"
	baselineResultsStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	indicatorsStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton Evaluator
)

// Singleton returns the Evaluator instance.
func Singleton() Evaluator {
	once.Do(func() {
		singleton = New(baselineResultsStore.Singleton(), baselinesStore.Singleton(), indicatorsStore.Singleton())
	})
	return singleton
}
