package enrichanddetect

import (
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Loop combines periodically (every hour) runs enrichment and detection.
type Loop interface {
	Start()
	ShortCircuit()
	Stop()
}

// NewLoop returns a new instance of a Loop.
func NewLoop(enricherAndDetector EnricherAndDetector, deployments datastore.DataStore) Loop {
	return newLoopWithDuration(enricherAndDetector, deployments, time.Hour)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(enricherAndDetector EnricherAndDetector, deployments datastore.DataStore, tickerDuration time.Duration) Loop {
	return &loopImpl{
		tickerDuration:      tickerDuration,
		stopChan:            concurrency.NewSignal(),
		shortChan:           make(chan struct{}),
		enricherAndDetector: enricherAndDetector,
		deployments:         deployments,
	}
}
