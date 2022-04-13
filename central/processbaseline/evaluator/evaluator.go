package evaluator

import (
	baselinesStore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	baselineResultsStore "github.com/stackrox/stackrox/central/processbaselineresults/datastore"
	indicatorsStore "github.com/stackrox/stackrox/central/processindicator/datastore"
	"github.com/stackrox/stackrox/generated/storage"
)

// An Evaluator evaluates process baselines, and stores their cached results.
//go:generate mockgen-wrapper
type Evaluator interface {
	EvaluateBaselinesAndPersistResult(deployment *storage.Deployment) (violatingProcesses []*storage.ProcessIndicator, err error)
}

// New returns a new evaluator.
func New(baselineResults baselineResultsStore.DataStore, baselines baselinesStore.DataStore, indicators indicatorsStore.DataStore) Evaluator {
	return &evaluator{
		baselineResults: baselineResults,
		baselines:       baselines,
		indicators:      indicators,
	}
}
