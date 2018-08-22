package enrichanddetect

import (
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type loopImpl struct {
	ticker    *time.Ticker
	shortChan chan struct{}
	stopChan  chan struct{}

	enricherAndDetector EnricherAndDetector
	deployments         datastore.DataStore
}

// Start starts the enrich and detect loop.
func (e *loopImpl) Start() {
	e.ticker = time.NewTicker(time.Hour)
	go e.loop()
}

// Stop stops the enrich and detect loop.
func (e *loopImpl) Stop() {
	e.stopChan <- struct{}{}
}

func (e *loopImpl) ShortCircuit() {
	e.shortChan <- struct{}{}
}

func (e *loopImpl) loop() {
	defer e.ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-e.shortChan:
		case <-e.ticker.C:
			e.enrichAndDetectAllDeployments()
		}
	}
}

func (e *loopImpl) enrichAndDetectAllDeployments() {
	deployments, err := e.deployments.GetDeployments()
	if err != nil {
		log.Error("unable to load deployments for reprocess loop: ", err)
		return
	}
	for _, deployment := range deployments {
		e.enricherAndDetector.EnrichAndDetect(deployment)
	}
}
