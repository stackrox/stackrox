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
	select {
	case <-kill.Done():
		return nil, errors.New("scraped stopped due ot received kill command")
	case <-scrape.Stopped().Done():
		if err := scrape.Stopped().Err(); err != nil {
			return nil, fmt.Errorf("scrape failed: %s", err)
		}
	}

	return scrape.GetResults(), nil
}
