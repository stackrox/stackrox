package compliance

import (
	"github.com/stackrox/rox/pkg/set"
)

type scrapeState struct {
	deploymentName string
	remainingHosts set.StringSet
}

func newScrapeState(name string, expectedHosts []string) *scrapeState {
	expectedHostsSet := set.NewStringSet()
	for _, host := range expectedHosts {
		expectedHostsSet.Add(host)
	}
	return &scrapeState{
		deploymentName: name,
		remainingHosts: expectedHostsSet,
	}
}
