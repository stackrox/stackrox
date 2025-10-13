package scrape

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

type controllerImpl struct {
	stoppedSig   concurrency.ReadOnlyErrorSignal
	scrapes      map[string]*scrapeImpl
	scrapesMutex sync.RWMutex

	msgInjector common.MessageInjector
}

func (s *controllerImpl) sendStartScrapeMsg(ctx concurrency.Waitable, scrape *scrapeImpl) error {
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ScrapeCommand{
			ScrapeCommand: &central.ScrapeCommand{
				ScrapeId: scrape.scrapeID,
				Command: &central.ScrapeCommand_StartScrape{
					StartScrape: &central.StartScrape{
						Hostnames: scrape.expectedHosts.AsSlice(),
						Standards: scrape.GetStandardIDs(),
					},
				},
			},
		},
	}
	return s.msgInjector.InjectMessage(ctx, msg)
}

func (s *controllerImpl) sendKillScrapeMsg(ctx concurrency.Waitable, scrape *scrapeImpl) error {
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ScrapeCommand{
			ScrapeCommand: &central.ScrapeCommand{
				ScrapeId: scrape.scrapeID,
				Command: &central.ScrapeCommand_KillScrape{
					KillScrape: &central.KillScrape{},
				},
			},
		},
	}
	return s.msgInjector.InjectMessage(ctx, msg)
}

func (s *controllerImpl) RunScrape(expectedHosts set.StringSet, kill concurrency.Waitable, standardIDs []string) (map[string]*compliance.ComplianceReturn, error) {
	// If no hosts need to be scraped, just bounce, otherwise we will be waiting for nothing.
	if expectedHosts.Cardinality() == 0 {
		return make(map[string]*compliance.ComplianceReturn), nil
	}

	// Create the scrape and register it so it can be updated.
	scrape := newScrape(expectedHosts, set.NewStringSet(standardIDs...))

	concurrency.WithLock(&s.scrapesMutex, func() {
		s.scrapes[scrape.scrapeID] = scrape
	})
	defer concurrency.WithLock(&s.scrapesMutex, func() {
		delete(s.scrapes, scrape.scrapeID)
	})

	if err := s.sendStartScrapeMsg(kill, scrape); err != nil {
		return nil, err
	}

	defer func() {
		if err := s.sendKillScrapeMsg(scrape.Stopped(), scrape); err != nil {
			log.Errorf("tried to kill scrape %s but failed: %v", scrape.scrapeID, err)
		}
	}()

	// Either receive a kill, or wait for the scrape to finish.
	var err error
	select {
	case <-kill.Done():
		err = errors.New("scrape stopped due to received kill command")
	case <-s.stoppedSig.Done():
		err = errors.Wrap(s.stoppedSig.Err(), "scrape stopped as sensor connection was terminated")
	case <-scrape.Stopped().Done():
		if scrapeErr := scrape.Stopped().Err(); scrapeErr != nil {
			err = errors.Wrap(scrapeErr, "scrape failed")
		}
	}

	return scrape.GetResults(), err
}

func (s *controllerImpl) getScrape(scrapeID string, remove bool) *scrapeImpl {
	s.scrapesMutex.RLock()
	defer s.scrapesMutex.RUnlock()

	scrape := s.scrapes[scrapeID]
	if remove {
		delete(s.scrapes, scrapeID)
	}
	return scrape
}

// AcceptUpdate forwards the update to a matching registered scrape.
func (s *controllerImpl) ProcessScrapeUpdate(update *central.ScrapeUpdate) error {
	scrape := s.getScrape(update.GetScrapeId(), update.GetScrapeKilled() != nil)
	if scrape == nil {
		return fmt.Errorf("received update for invalid scrape ID %q: %+v", update.GetScrapeId(), update)
	}

	scrape.AcceptUpdate(update)
	return nil
}
