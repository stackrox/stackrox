package evaluator

import (
	indicatorsStore "github.com/stackrox/rox/central/processindicator/datastore"
	whitelistsStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	whitelistResultsStore "github.com/stackrox/rox/central/processwhitelistresults/datastore"
	"github.com/stackrox/rox/generated/storage"
)

// An Evaluator evaluates process whitelists, and stores their cached results.
//go:generate mockgen-wrapper Evaluator
type Evaluator interface {
	EvaluateWhitelistsAndPersistResult(deployment *storage.Deployment) (violatingProcesses []*storage.ProcessIndicator, err error)
}

// New returns a new evaluator.
func New(whitelistResults whitelistResultsStore.DataStore, whitelists whitelistsStore.DataStore, indicators indicatorsStore.DataStore) Evaluator {
	return &evaluator{
		whitelistResults: whitelistResults,
		whitelists:       whitelists,
		indicators:       indicators,
	}
}
