package scrape

import (
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
)

// Controller starts and stops scrapes.
type Controller interface {
	ProcessScrapeUpdate(update *central.ScrapeUpdate) error
	RunScrape(expectedHosts set.StringSet, kill concurrency.Waitable, standardIDs []string) (map[string]*compliance.ComplianceReturn, error)
}

// NewController returns a new instance of a Controller.
func NewController(msgInjector common.MessageInjector, stoppedSig concurrency.ReadOnlyErrorSignal) Controller {
	return &controllerImpl{
		stoppedSig:  stoppedSig,
		scrapes:     make(map[string]*scrapeImpl),
		msgInjector: msgInjector,
	}
}

//go:generate mockgen-wrapper
