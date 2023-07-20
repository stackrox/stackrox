package billingmetrics

import (
	"time"

	"github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

const period = 1 * time.Hour

var (
	stop concurrency.Signal
	log  = logging.LoggerForModule()
)

func gather() {
	ids, err := getClusterIDs()
	if err != nil {
		log.Debug("Failed to get cluster IDs for billing metrics snapshot: ", err)
		return
	}
	newMetrics := clustermetrics.CutMetrics(ids)
	// Store the average values to smooth short (< 2 periods) peaks and drops.
	if err := checkIn(average(previousMetrics, newMetrics)); err != nil {
		log.Debug("Failed to store a billing metrics snapshot: ", err)
	}
	previousMetrics = newMetrics
}

func run() {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	gather()
	for {
		select {
		case <-ticker.C:
			gather()
		case <-stop.Done():
			stop.Reset()
			return
		}
	}
}

// Schedule initiates periodic data injections to the database with the
// collected billing metrics.
func Schedule() {
	go run()
}

// Stop stops the scheduled timer
func Stop() {
	stop.Signal()
}
