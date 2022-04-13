package evaluator

import (
	baselinesStore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	baselineResultsStore "github.com/stackrox/stackrox/central/processbaselineresults/datastore"
	indicatorsStore "github.com/stackrox/stackrox/central/processindicator/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
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
