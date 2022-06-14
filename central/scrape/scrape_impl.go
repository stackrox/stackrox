package scrape

import (
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

type scrapeImpl struct {
	// immutable, not protected by mutex
	scrapeID      string
	expectedHosts set.StringSet
	creationTime  time.Time
	standardIDs   set.StringSet
	////////////////////////////////////

	lock    sync.RWMutex
	results map[string]*compliance.ComplianceReturn

	stopped concurrency.ErrorSignal
}

func (s *scrapeImpl) GetScrapeID() string {
	return s.scrapeID
}

func (s *scrapeImpl) GetExpectedHosts() []string {
	return s.expectedHosts.AsSlice()
}

func (s *scrapeImpl) GetCreationTime() time.Time {
	return s.creationTime
}

func (s *scrapeImpl) GetStandardIDs() []string {
	return s.standardIDs.AsSlice()
}

func (s *scrapeImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stopped
}

func (s *scrapeImpl) GetResults() map[string]*compliance.ComplianceReturn {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ret := make(map[string]*compliance.ComplianceReturn, len(s.results))
	for id, cr := range s.results {
		ret[id] = cr
	}
	return ret
}

// Update the scrape with a new message from sensor.
func (s *scrapeImpl) AcceptUpdate(update *central.ScrapeUpdate) {
	s.lock.Lock()
	defer s.lock.Unlock()

	switch update.Update.(type) {
	case *central.ScrapeUpdate_ScrapeStarted:
		s.acceptStart(update.GetScrapeStarted())
	case *central.ScrapeUpdate_ComplianceReturn:
		s.acceptComplianceReturn(update.GetComplianceReturn())
	case *central.ScrapeUpdate_ScrapeKilled:
		s.acceptKill(update.GetScrapeKilled())
	default:
		log.Errorf("unrecognized scrape update: %s", proto.MarshalTextString(update))
	}
}

func (s *scrapeImpl) acceptStart(started *central.ScrapeStarted) {
	// If the scrape is already stopped, just log an error.
	if s.stopped.IsDone() {
		log.Errorf("scrape %s received a start update after being stopped", s.scrapeID)
		return
	}

	// If it failed to start, close everything.
	if started.GetErrorMessage() != "" {
		s.stopped.SignalWithError(fmt.Errorf("failed to start: %s", started.GetErrorMessage()))
	}
}

func (s *scrapeImpl) acceptComplianceReturn(cr *compliance.ComplianceReturn) {
	// If the scrape is already stopped, just log an error.
	if s.stopped.IsDone() {
		log.Errorf("scrape %s received an update for node %s after being stopped", s.scrapeID, cr.GetNodeName())
		return
	}

	// Check that the update is from an expected host. If not, just log an error and return.
	if !s.expectedHosts.Contains(cr.GetNodeName()) {
		log.Errorf("scrape %s received results from unexpected host: %s", s.scrapeID, cr.GetNodeName())
		return
	}

	// Check that we did not already received an update for the host.
	if _, exists := s.results[cr.GetNodeName()]; exists {
		log.Errorf("scrape %s received multiple results for host %s", cr.GetNodeName(), s.scrapeID)
		return
	}

	// Add the update to the results.
	s.results[cr.GetNodeName()] = cr
}

func (s *scrapeImpl) acceptKill(killed *central.ScrapeKilled) {
	// If the scrape is already stopped, just log an error.
	if s.stopped.IsDone() {
		log.Errorf("scrape %s received a kill update after being stopped", s.scrapeID)
		return
	}

	// If the kill is a result of an error (for instance a kill command from central), add the error to the signal.
	// Otherwise, just signal that we finished.
	if killed.GetErrorMessage() != "" {
		s.stopped.SignalWithError(errors.New(killed.GetErrorMessage()))
		return
	}

	// Check that we received the results we expected.
	// Since we do not allow repeated results, or unexpected results, we can just check that counts.
	if s.expectedHosts.Cardinality() != len(s.results) {
		s.stopped.SignalWithError(fmt.Errorf("scrape %s received kill with %d results outstanding", s.scrapeID, s.expectedHosts.Cardinality()-len(s.results)))
		return
	}

	// Everything is gucci baby.
	s.stopped.Signal()
}
