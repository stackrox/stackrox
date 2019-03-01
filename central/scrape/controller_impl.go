package scrape

import (
	"errors"
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/scrape/sensor/accept"
	"github.com/stackrox/rox/central/scrape/sensor/emit"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
)

type controllerImpl struct {
	emitter  emit.Emitter
	accepter accept.Accepter
}

func (s *controllerImpl) RunScrape(domain framework.ComplianceDomain, kill concurrency.Waitable) (map[string]*compliance.ComplianceReturn, error) {
	// Build a set of the expected nodes.
	expectedHosts := set.NewStringSet()
	for _, node := range framework.Nodes(domain) {
		expectedHosts.Add(node.GetName())
	}
	// If no hosts need to be scraped, just bounce, otherwise we will be waiting for nothing.
	if expectedHosts.Cardinality() == 0 {
		return make(map[string]*compliance.ComplianceReturn), nil
	}

	// Create the scrape and register it so it can be updated.
	scrape := newScrape(domain.Cluster().ID(), expectedHosts)

	// To start we need to register to accept updates. We do this before sending the message to
	// sensor so that we don't miss anything due to a race condition.
	s.accepter.AddFragment(scrape)
	defer s.accepter.RemoveFragment(scrape)

	// And we need to successfully send a signal to the sensor to begin.
	if err := s.emitter.StartScrape(scrape.GetClusterID(), scrape.GetScrapeID(), scrape.GetExpectedHosts()); err != nil {
		return nil, err
	}
	defer func() {
		if err := s.emitter.KillScrape(scrape.GetClusterID(), scrape.GetScrapeID()); err != nil {
			log.Errorf("tried to kill scrape but failed: %s", err)
		}
	}()

	// Either receive a kill, or wait for the scrape to finish.
	var err error
	select {
	case <-kill.Done():
		err = errors.New("scrape stopped due to received kill command")
	case <-scrape.Stopped().Done():
		if scrapeErr := scrape.Stopped().Err(); scrapeErr != nil {
			err = fmt.Errorf("scrape failed: %s", scrapeErr)
		}
	}

	return scrape.GetResults(), err
}
