package aggregator

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// ProcessAggregator combines the incoming process indicators and generates updates for each <deployment, container>.
//
//go:generate mockgen-wrapper
type ProcessAggregator interface {
	Add(indicators []*storage.ProcessIndicator)
	GetAndPrune(imageScanned func(string) bool, deploymentIDSet set.StringSet) map[string][]*ProcessUpdate
	RefreshDeployment(deployment *storage.Deployment)
}

// NewAggregator tracks and preprocesses new process indicators.
func NewAggregator() ProcessAggregator {
	return &aggregatorImpl{cache: make(map[string]map[string]*ProcessUpdate)}
}
