package scrape

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
)

// Scrape represents an instance of scraping info from the hosts in a cluster.
// A scrape is stopped when a kill update is received from sensor, or when all expected results are returned.
type Scrape interface {
	GetScrapeID() string
	GetExpectedHosts() []string
	GetCreationTime() time.Time

	Stopped() concurrency.ReadOnlyErrorSignal
	GetResults() map[string]*compliance.ComplianceReturn
}

func newScrape(expectedHosts set.StringSet, standardIDs set.StringSet) *scrapeImpl {
	return &scrapeImpl{
		scrapeID:      uuid.NewV4().String(),
		expectedHosts: expectedHosts,
		creationTime:  time.Now(),
		standardIDs:   standardIDs,

		results: make(map[string]*compliance.ComplianceReturn),

		stopped: concurrency.NewErrorSignal(),
	}
}
