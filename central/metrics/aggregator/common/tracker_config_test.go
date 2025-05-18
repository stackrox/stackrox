package common

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func TestMakeTrackerConfig(t *testing.T) {
	tracker := MakeTrackerConfig("test", "test", testLabelOrder)
	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.periodCh)

	mle := tracker.GetMetricLabelExpressions()
	assert.Nil(t, mle)
}

func TestTrackerConfig_Reconfigure(t *testing.T) {

	t.Run("test 0 period", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelOrder)

		assert.NoError(t, tracker.Reconfigure(nil, nil, 0))
		assert.Nil(t, tracker.GetMetricLabelExpressions())
	})

	t.Run("test with good test configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelOrder)
		assert.NoError(t, tracker.Reconfigure(nil, makeTestMetricLabels(t), 42*time.Hour))
		mle := tracker.GetMetricLabelExpressions()
		assert.NotNil(t, mle)
		select {
		case period := <-tracker.GetPeriodCh():
			assert.Equal(t, 42*time.Hour, period)
		default:
			assert.Fail(t, "should have period configured")
		}
		assert.Equal(t, makeTestMetricLabelExpressions(t), mle)
	})

	t.Run("test with initial bad configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelOrder)
		err := tracker.Reconfigure(nil, map[string]*storage.PrometheusMetricsConfig_LabelExpressions{
			" ": nil,
		}, 11*time.Hour)

		assert.ErrorIs(t, err, errox.InvalidArgs)
		assert.Equal(t, `invalid arguments: invalid metric name " ": bad characters`, err.Error())

		assert.Nil(t, tracker.GetMetricLabelExpressions())
		select {
		case period := <-tracker.GetPeriodCh():
			assert.Fail(t, "period configured: %v", period)
		default:
		}
	})

	t.Run("test with bad reconfiguration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelOrder)
		assert.NoError(t, tracker.Reconfigure(nil, makeTestMetricLabels(t), 42*time.Hour))

		err := tracker.Reconfigure(nil, map[string]*storage.PrometheusMetricsConfig_LabelExpressions{
			"m1": {
				LabelExpressions: map[string]*storage.PrometheusMetricsConfig_LabelExpressions_Expressions{
					"label1": nil,
				},
			},
		}, 11*time.Hour)
		assert.ErrorIs(t, err, errox.InvalidArgs)
		assert.Equal(t, `invalid arguments: unknown label "label1" for metric "m1"`, err.Error())

		mle := tracker.GetMetricLabelExpressions()
		assert.NotNil(t, mle)
		select {
		case period := <-tracker.GetPeriodCh():
			assert.Equal(t, 42*time.Hour, period)
		default:
			assert.Fail(t, "no period in the channel")
		}
		assert.Equal(t, makeTestMetricLabelExpressions(t), mle)
	})
}
