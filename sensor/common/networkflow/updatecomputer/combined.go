package updatecomputer

import (
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

type Combined struct {
	main  UpdateComputer
	other UpdateComputer
}

func NewCombined(main UpdateComputer, other UpdateComputer) *Combined {
	return &Combined{
		main:  main,
		other: other,
	}
}

func (c *Combined) ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) ([]*storage.NetworkFlow, error) {
	ul, _ := c.other.ComputeUpdatedConns(current)
	uc, err := c.main.ComputeUpdatedConns(current)
	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("connections", "legacy").Add(float64(len(ul)))
	flowMetrics.NumUpdatesSentToCentralCurrent.WithLabelValues("connections", "legacy").Set(float64(len(ul)))
	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("connections", "categorized").Add(float64(len(uc)))
	flowMetrics.NumUpdatesSentToCentralCurrent.WithLabelValues("connections", "categorized").Set(float64(len(uc)))

	return uc, errors.Wrap(err, "main computer")
}
func (c *Combined) ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	ul := c.other.ComputeUpdatedEndpoints(current)
	uc := c.main.ComputeUpdatedEndpoints(current)

	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("endpoints", "legacy").Add(float64(len(ul)))
	flowMetrics.NumUpdatesSentToCentralCurrent.WithLabelValues("endpoints", "legacy").Set(float64(len(ul)))
	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("endpoints", "categorized").Add(float64(len(uc)))
	flowMetrics.NumUpdatesSentToCentralCurrent.WithLabelValues("endpoints", "categorized").Set(float64(len(uc)))
	return uc
}
func (c *Combined) ComputeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
	ul := c.other.ComputeUpdatedProcesses(current)
	uc := c.main.ComputeUpdatedProcesses(current)

	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("processes", "legacy").Add(float64(len(ul)))
	flowMetrics.NumUpdatesSentToCentralCurrent.WithLabelValues("processes", "legacy").Set(float64(len(ul)))
	flowMetrics.NumUpdatesSentToCentral.WithLabelValues("processes", "categorized").Add(float64(len(uc)))
	flowMetrics.NumUpdatesSentToCentralCurrent.WithLabelValues("processes", "categorized").Set(float64(len(uc)))
	return uc
}

// UpdateState covers state management - each implementation handles its own state updates
func (c *Combined) UpdateState(currentConns map[indicator.NetworkConn]timestamp.MicroTS, currentEndpoints map[indicator.ContainerEndpoint]timestamp.MicroTS, currentProcesses map[indicator.ProcessListening]timestamp.MicroTS) {
	c.other.UpdateState(currentConns, currentEndpoints, currentProcesses)
	c.main.UpdateState(currentConns, currentEndpoints, currentProcesses)
}

// ResetState resets all internal state (used when clearing historical data)
func (c *Combined) ResetState() {
	c.other.ResetState()
	c.main.ResetState()
}

// PeriodicCleanup should be run periodically to clean up the temporal data.
func (c *Combined) PeriodicCleanup(now time.Time, cleanupInterval time.Duration) {
	c.other.PeriodicCleanup(now, cleanupInterval)
	c.main.PeriodicCleanup(now, cleanupInterval)
}

// GetStateMetrics returns metric values about internal state size for monitoring
func (c *Combined) GetStateMetrics() map[string]map[string]int {
	c.other.GetStateMetrics()
	return c.main.GetStateMetrics()
}
func (c *Combined) RecordSizeMetrics(_ string, gv *prometheus.GaugeVec) {
	c.other.RecordSizeMetrics("legacy", gv)
	c.main.RecordSizeMetrics("categorized", gv)
}
