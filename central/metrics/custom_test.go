package metrics

import (
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestRegisterCustomAggregatedMetric(t *testing.T) {
	assert.NoError(t, RegisterCustomAggregatedMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))
	assert.NoError(t, RegisterCustomAggregatedMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))

	SetCustomAggregatedCount("test", map[string]string{"Test1": "value1", "Test2": "value2"}, 42)

	metric, ok := customAggregatedMetrics.Load("test")
	if assert.True(t, ok) {
		metricValue := &dto.Metric{}
		err := metric.(*metricRecord).GaugeVec.WithLabelValues("value1", "value2").Write(metricValue)
		assert.NoError(t, err)
		assert.Equal(t, float64(42), metricValue.GetGauge().GetValue())
	}

	assert.True(t, UnregisterCustomAggregatedMetric("test"))
	assert.False(t, UnregisterCustomAggregatedMetric("test"))
}
