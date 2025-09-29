package metrics

import (
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func getMetricValue(_ *testing.T, registry CustomRegistry, metricName string) (float64, error) {
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

func TestMakeCustomRegistry(t *testing.T) {
	cr1, err := GetCustomRegistry("user1")
	assert.NoError(t, err)
	cr2, err := GetCustomRegistry("user2")
	assert.NoError(t, err)
	assert.NotSame(t, cr1, cr2)

	assert.NoError(t, cr1.RegisterMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))
	assert.NoError(t, cr1.RegisterMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))
	assert.NoError(t, cr2.RegisterMetric("test", "test", 10*time.Minute, []string{"Test1", "Test2"}))

	cr1.SetTotal("test", map[string]string{"Test1": "value1", "Test2": "value2"}, 42)
	cr2.SetTotal("test", map[string]string{"Test1": "value1", "Test2": "value2"}, 24)

	value, err := getMetricValue(t, cr1, "test")
	assert.NoError(t, err)
	assert.Equal(t, float64(42), value)

	value, err = getMetricValue(t, cr2, "test")
	assert.NoError(t, err)
	assert.Equal(t, float64(24), value)

	assert.True(t, cr1.UnregisterMetric("test"))
	assert.False(t, cr1.UnregisterMetric("test"))
	assert.True(t, cr2.UnregisterMetric("test"))
}

func TestCustomRegistry_Reset(t *testing.T) {
	cr, err := GetCustomRegistry("user1")
	assert.NoError(t, err)
	assert.NoError(t, cr.RegisterMetric("test1", "test", 10*time.Minute, []string{"Test1", "Test2"}))
	assert.NoError(t, cr.RegisterMetric("test2", "test", 10*time.Minute, []string{"Test1", "Test2"}))
	cr.SetTotal("test1", map[string]string{"Test1": "value1", "Test2": "value2"}, 42)
	cr.SetTotal("test2", map[string]string{"Test1": "value1", "Test2": "value2"}, 24)

	cr.Reset("test1")

	value, err := getMetricValue(t, cr, "test1")
	assert.NoError(t, err)
	assert.Equal(t, float64(0), value)

	value, err = getMetricValue(t, cr, "test2")
	assert.NoError(t, err)
	assert.Equal(t, float64(24), value)
}

func TestDeleteCustomRegistry(t *testing.T) {
	cr1, err := GetCustomRegistry("user1")
	assert.NoError(t, err)
	cr2, err := GetCustomRegistry("user2")
	assert.NoError(t, err)
	_ = cr1.RegisterMetric("test", "test", time.Hour, []string{"Test1", "Test2"})
	_ = cr2.RegisterMetric("test", "test", time.Hour, []string{"Test1", "Test2"})
	cr1.SetTotal("test", map[string]string{"Test1": "value1", "Test2": "value2"}, 42)
	cr2.SetTotal("test", map[string]string{"Test1": "value1", "Test2": "value2"}, 24)

	DeleteCustomRegistry("user1")

	value, err := getMetricValue(t, cr1, "test")
	assert.Error(t, err)
	assert.Equal(t, float64(0), value)

	cr1, err = GetCustomRegistry("user1")
	assert.NoError(t, err)
	value, err = getMetricValue(t, cr1, "test")
	assert.Error(t, err)
	assert.Equal(t, float64(0), value)

	value, err = getMetricValue(t, cr2, "test")
	assert.NoError(t, err)
	assert.Equal(t, float64(24), value)

	assert.NotPanics(t, func() {
		DeleteCustomRegistry("user1")
		DeleteCustomRegistry("user1")
		DeleteCustomRegistry("user100")
	})
}

func TestErrTooMany(t *testing.T) {
	// Saturate.
	for i := range maxCustomRegistries {
		_, err := GetCustomRegistry("user" + strconv.Itoa(i))
		assert.NoError(t, err)
	}
	_, err := GetCustomRegistry("user" + strconv.Itoa(maxCustomRegistries))
	assert.ErrorIs(t, err, ErrTooMany)
	DeleteCustomRegistry("user0")
	_, err = GetCustomRegistry("user0")
	assert.NoError(t, err)
	// Cleanup.
	for i := range maxCustomRegistries {
		DeleteCustomRegistry("user" + strconv.Itoa(i))
	}
}
