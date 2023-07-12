package gatherers

import (
	"context"
	"time"

	bmetrics "github.com/stackrox/rox/central/billingmetrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

func updateMaxima(ctx context.Context, clusters []*data.ClusterInfo) {
	var totalMilliCores, totalNodes int
	for _, c := range clusters {
		totalNodes += len(c.Nodes)
		for _, n := range c.Nodes {
			if n != nil && n.TotalResources != nil {
				totalMilliCores += n.TotalResources.MilliCores
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
