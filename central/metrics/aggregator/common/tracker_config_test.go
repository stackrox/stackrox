package common

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func nilGatherFunc(_ context.Context) iter.Seq[Finding] {
	return func(func(Finding) bool) {}
}

func makeTestGatherFunc(data []map[Label]string) func(_ context.Context) iter.Seq[Finding] {
	return func(_ context.Context) iter.Seq[Finding] {
		return func(yield func(Finding) bool) {
			for _, datum := range data {
				if !yield(func(label Label) string {
					return datum[label]
				}) {
					return
				}
			}
		}
	}
}

func TestMakeTrackerConfig(t *testing.T) {
	tracker := MakeTrackerConfig("test", "test", testLabelOrder, nilGatherFunc)
	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.periodCh)

	mle := tracker.GetMetricLabelExpressions()
	assert.Nil(t, mle)
}

func TestTrackerConfig_Reconfigure(t *testing.T) {

	t.Run("test 0 period", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelOrder, nilGatherFunc)

		assert.NoError(t, tracker.Reconfigure(nil, nil, 0))
		assert.Nil(t, tracker.GetMetricLabelExpressions())
	})

	t.Run("test with good test configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelOrder, nilGatherFunc)
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
		tracker := MakeTrackerConfig("test", "test", testLabelOrder, nilGatherFunc)
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
		tracker := MakeTrackerConfig("test", "test", testLabelOrder, nilGatherFunc)
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

func TestMakeTrackFunc(t *testing.T) {
	type myDS struct{}
	result := make(map[string][]*record)
	cfg := MakeTrackerConfig("test", "test",
		testLabelOrder,
		makeTestGatherFunc([]map[Label]string{
			{
				"Severity":  "CRITICAL",
				"Cluster":   "cluster 1",
				"Namespace": "ns 1",
			}, {
				"Severity":  "HIGH",
				"Cluster":   "cluster 2",
				"Namespace": "ns 2",
			},
			{
				"Severity":  "LOW",
				"Cluster":   "cluster 3",
				"Namespace": "ns 3",
			},
			{
				"Severity":  "CRITICAL",
				"Cluster":   "cluster 1",
				"Namespace": "ns 3",
			},
			{
				"Severity":  "LOW",
				"Cluster":   "cluster 5",
				"Namespace": "ns 3",
			},
		}),
	)
	track := MakeTrackFunc(
		cfg,
		func() MetricLabelExpressions {
			return makeTestMetricLabelExpressions(t)
		},
		func(metricName string, labels prometheus.Labels, total int) {
			result[metricName] = append(result[metricName], &record{labels, total})
		},
	)

	track(context.Background())

	if assert.Len(t, result, 2) &&
		assert.Contains(t, result, "TestMakeTrackFunc_metric1") &&
		assert.Contains(t, result, "TestMakeTrackFunc_metric2") {

		assert.ElementsMatch(t, result["TestMakeTrackFunc_metric1"],
			[]*record{
				{prometheus.Labels{
					"Severity": "CRITICAL",
					"Cluster":  "cluster 1",
				}, 2},
				{prometheus.Labels{
					"Severity": "HIGH",
					"Cluster":  "cluster 2",
				}, 1},
			})

		assert.ElementsMatch(t, result["TestMakeTrackFunc_metric2"],
			[]*record{
				{prometheus.Labels{
					"Namespace": "ns 1",
				}, 1},
				{prometheus.Labels{
					"Namespace": "ns 2",
				}, 1},
				{prometheus.Labels{
					"Namespace": "ns 3",
				}, 3},
			})
	}
}
