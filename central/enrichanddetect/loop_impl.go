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
	stopped        concurrency.Signal

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
	l.stopped.Wait()
}

func (l *loopImpl) ShortCircuit() {
	select {
	case l.shortChan <- struct{}{}:
	case <-l.stopped.Done():
	}
}

func (l *loopImpl) loop() {
	defer l.stopped.Signal()
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
		log.Errorf("unable to load deployments for reprocess loop: %v", err)
		return
	}
	for _, deployment := range deployments {
		if err := l.enricherAndDetector.EnrichAndDetect(deployment); err != nil {
			log.Errorf("Failed to enrich deployment %s: %v", deployment.GetId(), err)
		}
	}
}
