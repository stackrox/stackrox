package billingmetrics

import (
	"context"
	"time"

	bmetrics "github.com/stackrox/rox/central/billingmetrics"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
)

const period = 1 * time.Hour

var (
	sig concurrency.Signal
	log = logging.LoggerForModule()

	previousMetrics = clustermetrics.BillingMetrics{}
)

func gather() {
	newMetrics := clustermetrics.CutMetrics()
	{
		average := newMetrics
		average.TotalNodes = (average.TotalNodes + previousMetrics.TotalNodes) / 2
		average.TotalMilliCores = (average.TotalMilliCores + previousMetrics.TotalMilliCores) / 2
		checkIn(average)
	}
	previousMetrics = newMetrics
}

// Schedule starts a periodic data gathering from the secured clusters, updating
// the maximus store with the total numbers of secured nodes and millicores.
func Schedule() {
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		gather()
		for {
			select {
			case <-ticker.C:
				gather()
			case <-sig.Done():
				sig.Reset()
				return
			}
		}
	}()
}

// Stop stops the scheduled timer
func Stop() {
	sig.Signal()
}

func checkIn(metrics clustermetrics.BillingMetrics) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	if _, err := bmetrics.Singleton().PutMetrics(ctx, &v1.BillingMetricsInsertRequest{
		Ts: protoconv.ConvertTimeToTimestamp(time.Now()),
		Metrics: &v1.SecuredResourcesMetrics{
			Nodes:      int32(metrics.TotalNodes),
			Millicores: int32(metrics.TotalMilliCores)},
	}); err != nil {
		log.Errorf("Error inserting billing metrics: %v", err)
	}
}
