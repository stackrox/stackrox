package billingmetrics

import (
	"context"
	"time"

	bmetrics "github.com/stackrox/rox/central/billingmetrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

var (
	log = logging.LoggerForModule()
)

func updateMaxima(ctx context.Context, clusters []*data.ClusterInfo) {
	var totalMilliCores, totalNodes int
	for _, cluster := range clusters {
		if cluster.Sensor == nil || !cluster.Sensor.CurrentlyConnected {
			continue
		}
		totalNodes += len(cluster.Nodes)
		for _, node := range cluster.Nodes {
			if node != nil && node.TotalResources != nil {
				totalMilliCores += node.TotalResources.MilliCores
			}
		}
	}

	now := protoconv.ConvertTimeToTimestamp(time.Now())
	update := bmetrics.Singleton().PostMaximum
	if _, err := update(ctx, &v1.MaximumValueUpdateRequest{
		Metric: "nodes", Value: int32(totalNodes), Ts: now,
	}); err != nil {
		log.Errorf("Error updating the maximum total nodes count: %v", err)
	}
	if _, err := update(ctx, &v1.MaximumValueUpdateRequest{
		Metric: "millicores", Value: int32(totalMilliCores), Ts: now,
	}); err != nil {
		log.Errorf("Error updating the maximum total millicores count: %v", err)
	}
}
