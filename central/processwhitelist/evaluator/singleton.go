package evaluator

import (
	indicatorsStore "github.com/stackrox/rox/central/processindicator/datastore"
	whitelistsStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	whitelistResultsStore "github.com/stackrox/rox/central/processwhitelistresults/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton Evaluator
)

// Singleton returns the Evaluator instance.
func Singleton() Evaluator {
	once.Do(func() {
		singleton = New(whitelistResultsStore.Singleton(), whitelistsStore.Singleton(), indicatorsStore.Singleton())
	})
	return singleton
}
