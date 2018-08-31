package enrichanddetect

import (
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type loopImpl struct {
	tickerDuration time.Duration
	ticker         *time.Ticker
	shortChan      chan struct{}
	stopChan       concurrency.Signal

	enricherAndDetector EnricherAndDetector
	deployments         datastore.DataStore
}

// Start starts the enrich and detect loop.
func (l *loopImpl) Start() {
	l.ticker = time.NewTicker(l.tickerDuration)
	go l.loop()
}

// Stop stops the enrich and detect loop.
func (l *loopImpl) Stop() {
	l.stopChan.Signal()
}

func (l *loopImpl) ShortCircuit() {
	l.shortChan <- struct{}{}
}

func (l *loopImpl) loop() {
	defer l.ticker.Stop()

	for {
		select {
		case <-l.stopChan.Done():
			return
		case <-l.shortChan:
			l.enrichAndDetectAllDeployments()
		case <-l.ticker.C:
			l.enrichAndDetectAllDeployments()
		}
	}
}

func (l *loopImpl) enrichAndDetectAllDeployments() {
	deployments, err := l.deployments.GetDeployments()
	if err != nil {
		log.Error("unable to load deployments for reprocess loop: ", err)
		return
	}
	for _, deployment := range deployments {
		l.enricherAndDetector.EnrichAndDetect(deployment)
	}
}
