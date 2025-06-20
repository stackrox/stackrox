package metrics

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestMakeCustomRegistry(t *testing.T) {
	cr1 := MakeCustomRegistry()
	cr2 := MakeCustomRegistry()
	assert.NotSame(t, cr1, cr2)

	assert.NoError(t, cr1.RegisterMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))
	assert.NoError(t, cr1.RegisterMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))
	assert.NoError(t, cr2.RegisterMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))

	cr1.SetTotal("test", map[string]string{"Test1": "value1", "Test2": "value2"}, 42)
	cr2.SetTotal("test", map[string]string{"Test1": "value1", "Test2": "value2"}, 24)

	getMetricValue := func(registry CustomRegistry, metricName string) (float64, error) {
		metricValue := &dto.Metric{}
		gauge, ok := registry.(*customRegistry).gauges.Load(metricName)
		if !ok {
			return 0, errors.New("no such metric")
		}
		err := gauge.(*prometheus.GaugeVec).WithLabelValues("value1", "value2").Write(metricValue)
		if err != nil {
			return 0, err
		}
		return metricValue.GetGauge().GetValue(), nil
	}

	value, err := getMetricValue(cr1, "test")
	assert.NoError(t, err)
	assert.Equal(t, float64(42), value)

	value, err = getMetricValue(cr2, "test")
	assert.NoError(t, err)
	assert.Equal(t, float64(24), value)

	assert.True(t, cr1.UnregisterMetric("test"))
	assert.False(t, cr1.UnregisterMetric("test"))
	assert.True(t, cr2.UnregisterMetric("test"))
}
