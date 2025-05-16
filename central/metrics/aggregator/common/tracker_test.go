package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_vulnerabilityTracker_getMetricsConfig(t *testing.T) {
	tracker := MakeTracker("test", "test", testLabelsOrder)
	assert.NotNil(t, tracker)
	mc := tracker.GetMetricsConfig()
	assert.Nil(t, mc)

	t.Run("test 0 period", func(t *testing.T) {
		tracker.Reconfigure(nil, 0)
		mc = tracker.GetMetricsConfig()
		assert.Nil(t, mc)
	})

	t.Run("test with test configuration", func(t *testing.T) {
		metrics := MetricsConfig{
			"metric1": map[Label][]*Expression{
				"Severity": {
					{
						"=",
						"CRITICAL*",
					},
				},
			},
		}
		tracker.Reconfigure(metrics, 42*time.Hour)
		mc = tracker.GetMetricsConfig()
		assert.NotNil(t, mc)
		assert.Equal(t, 42*time.Hour, <-tracker.periodCh)
		assert.Equal(t, "=CRITICAL*", mc["metric1"]["Severity"][0].String())
	})
}
