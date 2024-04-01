package metrics

import (
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once           sync.Once
	metricsHandler *types.MetricsHandler
)

// Singleton returns an instance of the Prometheus metrics handler.
func Singleton() *types.MetricsHandler {
	once.Do(func() {
		metricsHandler = types.NewMetricsHandler(metrics.SensorSubsystem)
	})
	return metricsHandler
}
