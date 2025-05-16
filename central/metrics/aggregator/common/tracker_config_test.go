package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrackerConfig_GetMetricsConfig(t *testing.T) {
	tracker := MakeTrackerConfig("test", "test", testLabelsOrder)
	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.periodCh)

	mle := tracker.GetMetricLabelExpressions()
	assert.Nil(t, mle)

	t.Run("test 0 period", func(t *testing.T) {
		tracker.Reconfigure(nil, nil, 0)
		mle = tracker.GetMetricLabelExpressions()
		assert.Nil(t, mle)
	})

	t.Run("test with test configuration", func(t *testing.T) {
		metrics := MetricLabelExpressions{
			"metric1": map[Label][]*Expression{
				"Severity": {
					{
						"=",
						"CRITICAL*",
					},
				},
			},
		}
		tracker.Reconfigure(nil, metrics, 42*time.Hour)
		mle = tracker.GetMetricLabelExpressions()
		assert.NotNil(t, mle)
		assert.Equal(t, 42*time.Hour, <-tracker.periodCh)
		assert.Equal(t, "=CRITICAL*", mle["metric1"]["Severity"][0].String())
	})
}
