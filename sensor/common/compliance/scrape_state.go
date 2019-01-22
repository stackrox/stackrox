package compliance

import (
	"github.com/stackrox/rox/pkg/set"
)

type scrapeState struct {
	deploymentName string
	remainingNodes set.StringSet
}

func newScrapeState(name string, expectedNodes []string) *scrapeState {
	expectedNodesSet := set.NewStringSet()
	for _, host := range expectedNodes {
		expectedNodesSet.Add(host)
	}
	return &scrapeState{
		deploymentName: name,
		remainingNodes: expectedNodesSet,
	}
}
