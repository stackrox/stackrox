package compliance

import (
	"github.com/stackrox/stackrox/pkg/set"
)

type scrapeState struct {
	deploymentName string
	remainingNodes set.StringSet
	foundNodes     set.StringSet
	desiredNodes   int
}

func newScrapeState(name string, desiredNodes int, expectedNodes []string) *scrapeState {
	expectedNodesSet := set.NewStringSet()
	for _, host := range expectedNodes {
		expectedNodesSet.Add(host)
	}
	return &scrapeState{
		deploymentName: name,
		remainingNodes: expectedNodesSet,
		desiredNodes:   desiredNodes,
		foundNodes:     set.NewStringSet(),
	}
}
