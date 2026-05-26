package benchmark

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSumCounterDelta(t *testing.T) {
	before := map[string]float64{
		`rox_sensor_k8s_events{action="CREATE",resource="Pod"}`: 100,
	}
	after := map[string]float64{
		`rox_sensor_k8s_events{action="CREATE",resource="Pod"}`: 250,
	}
	require.Equal(t, 150.0, SumCounterDelta("rox_sensor_k8s_events", before, after))
}

func TestSumCounterDeltaFiltered(t *testing.T) {
	before := map[string]float64{
		`rox_sensor_sensor_events{resource="ProcessIndicator",type="total"}`:  50,
		`rox_sensor_sensor_events{resource="Deployment",type="total"}`:        200,
		`rox_sensor_sensor_events{resource="ProcessIndicator",type="unique"}`: 10,
	}
	after := map[string]float64{
		`rox_sensor_sensor_events{resource="ProcessIndicator",type="total"}`:  125,
		`rox_sensor_sensor_events{resource="Deployment",type="total"}`:        260,
		`rox_sensor_sensor_events{resource="ProcessIndicator",type="unique"}`: 15,
	}

	filter := map[string]string{
		"type":     "total",
		"resource": "ProcessIndicator",
	}
	require.Equal(t, 75.0, SumCounterDeltaFiltered("rox_sensor_sensor_events", before, after, filter))
}

func TestParseMetricFamilies(t *testing.T) {
	body := []byte(`# HELP rox_sensor_k8s_events Total number of Kubernetes resource events
# TYPE rox_sensor_k8s_events counter
rox_sensor_k8s_events{action="CREATE",resource="Pod"} 250
`)
	parsed, err := ParseMetricFamilies(body)
	require.NoError(t, err)
	require.Equal(t, map[string]float64{
		`rox_sensor_k8s_events{action="CREATE",resource="Pod"}`: 250,
	}, parsed)
}

func TestRatePerSec(t *testing.T) {
	require.Equal(t, 2.5, RatePerSec(150, 60))
	require.Equal(t, 0.0, RatePerSec(150, 0))
}
