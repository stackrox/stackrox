package billingmetrics

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

const period = 1 * time.Hour

var (
	log  = logging.LoggerForModule()
	stop = concurrency.NewSignal()
)

func gather(ctx context.Context) {
	ids, err := getClusterIDs(ctx)
	if err != nil {
		log.Debug("Failed to get cluster IDs for billing metrics snapshot: ", err)
		return
	}
	log.Debugf("Cutting billing metrics for %d clusters: %v", len(ids), ids)
	newMetrics := clustermetrics.CutMetrics(ids)
	// Store the average values to smooth short (< 2 periods) peaks and drops.
	if err := checkIn(average(previousMetrics, newMetrics)); err != nil {
		log.Debug("Failed to store a billing metrics snapshot: ", err)
	}
	previousMetrics = newMetrics
}

// GetCurrent returns the total values of last known metrics reported by known
// clusters.
func GetCurrent(ctx context.Context) *clustermetrics.BillingMetrics {
	ids, err := getClusterIDs(ctx)
	if err != nil {
		log.Debug("Failed to get cluster IDs for current billing metrics: ", err)
		return nil
	}
	log.Debugf("Cutting billing metrics for %d clusters: %v", len(ids), ids)
	return clustermetrics.FilterCurrent(ids)
}

func run() {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	gather(ctx)
	for {
		select {
		case <-ticker.C:
			gather(ctx)
		case <-stop.Done():
			cancel()
			log.Debug("Billing metrics reporting stopped")
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
