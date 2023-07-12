package billingmetrics

import (
	"context"
	"sync"
	"time"

	"github.com/stackrox/rox/central/telemetry/gatherers"
)

const period = 1 * time.Hour

var (
	once   sync.Once
	ticker *time.Ticker
	stop   chan bool
)

func gather() {
	if data := gatherers.Singleton().Gather(context.Background(), true, false); data != nil {
		updateMaxima(context.Background(), data.Clusters)
	}
}

// Schedule starts a periodic data gathering from the secured clusters, updating
// the maximus store with the total numbers of secured nodes and millicores.
func Schedule() {
	once.Do(func() {
		stop = make(chan bool, 1)
		ticker = time.NewTicker(period)
		go func() {
			gather()
		loop:
			for {
				select {
				case <-ticker.C:
					gather()
				case <-stop:
					ticker.Stop()
					break loop
				}
			}
		}()
	})
}

// Stop stops the scheduled timer
func Stop() {
	if ticker != nil {
		ticker.Stop()
		once = sync.Once{}
	}
}
