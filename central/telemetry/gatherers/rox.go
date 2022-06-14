package gatherers

import (
	"context"

	"github.com/stackrox/rox/pkg/telemetry/data"
)

// RoxGatherer is the top level gatherer that encompasses all the information we collect for Telemetry
type RoxGatherer struct {
	central *CentralGatherer
	cluster *ClusterGatherer
}

// newRoxGatherer creates and returns a RoxGatherer object
func newRoxGatherer(central *CentralGatherer, cluster *ClusterGatherer) *RoxGatherer {
	return &RoxGatherer{
		central: central,
		cluster: cluster,
	}
}

// Gather returns telemetry information about this Rox
func (c *RoxGatherer) Gather(ctx context.Context, pullFromSensors bool) *data.TelemetryData {
	return &data.TelemetryData{
		Central:  c.central.Gather(ctx),
		Clusters: c.cluster.Gather(ctx, pullFromSensors),
	}
}
