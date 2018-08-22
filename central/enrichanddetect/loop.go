package enrichanddetect

import (
	"github.com/stackrox/rox/central/deployment/datastore"
)

// Loop combines periodically (every hour) runs enrichment and detection.
type Loop interface {
	Start()
	ShortCircuit()
	Stop()
}

// NewLoop returns a new instance of a Loop.
func NewLoop(enricherAndDetector EnricherAndDetector, deployments datastore.DataStore) Loop {
	return &loopImpl{
		stopChan:            make(chan struct{}),
		shortChan:           make(chan struct{}),
		enricherAndDetector: enricherAndDetector,
		deployments:         deployments,
	}
}
