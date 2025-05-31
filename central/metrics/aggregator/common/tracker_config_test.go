package common

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

type testDataIndex int

func (t testDataIndex) Count() int {
	return 1
}

var testRegistry = prometheus.NewRegistry()

func nilGatherFunc(context.Context, *v1.Query, MetricLabelsExpressions) iter.Seq[testDataIndex] {
	return func(yield func(testDataIndex) bool) {}
}

func makeTestGatherFunc(data []map[Label]string) FindingGenerator[testDataIndex] {
	return func(context.Context, *v1.Query, MetricLabelsExpressions) iter.Seq[testDataIndex] {
		return func(yield func(testDataIndex) bool) {
			for i := range data {
				if !yield(testDataIndex(i)) {
					return
				}
			}
		}
	}
}

func TestMakeTrackerConfig(t *testing.T) {
	tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.periodCh)

	query, mle := tracker.GetMetricLabelExpressions()
	assert.Empty(t, query)
	assert.Nil(t, mle)
}

func TestTrackerConfig_Reconfigure(t *testing.T) {

	t.Run("test 0 period", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(testRegistry, "", nil, 0))
		query, mle := tracker.GetMetricLabelExpressions()
		assert.Equal(t, "", query.String())
		assert.Nil(t, mle)
	})

	t.Run("test query", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(testRegistry, `Cluster:name`, nil, 0))
		query, mle := tracker.GetMetricLabelExpressions()
		assert.Equal(t, "Cluster", query.GetBaseQuery().GetMatchFieldQuery().GetField())
		assert.Nil(t, mle)
	})

	t.Run("test bad query", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(testRegistry, `bad query?`, nil, 0))
	})

	t.Run("test with good test configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(testRegistry, "", makeTestMetricLabels(t), 42*time.Hour))
		_, mle := tracker.GetMetricLabelExpressions()
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
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		err := tracker.Reconfigure(testRegistry, "", map[string]*storage.PrometheusMetricsConfig_MetricLabels{
			" ": nil,
		}, 11*time.Hour)

		assert.ErrorIs(t, err, errox.InvalidArgs)
		assert.Equal(t, `invalid arguments: invalid metric name " ": bad characters`, err.Error())

		_, mle := tracker.GetMetricLabelExpressions()
		assert.Nil(t, mle)
		select {
		case period := <-tracker.GetPeriodCh():
			assert.Fail(t, "period configured: %v", period)
		default:
		}
	})

	t.Run("test with bad reconfiguration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(testRegistry, "", makeTestMetricLabels(t), 42*time.Hour))

		err := tracker.Reconfigure(testRegistry, "", map[string]*storage.PrometheusMetricsConfig_MetricLabels{
			"m1": {
				LabelExpressions: map[string]*storage.PrometheusMetricsConfig_MetricLabels_Expressions{
					"label1": nil,
				},
			},
		}, 11*time.Hour)
		assert.ErrorIs(t, err, errox.InvalidArgs)
		assert.Equal(t, `invalid arguments: unknown label "label1" for metric "m1"`, err.Error())

		_, mle := tracker.GetMetricLabelExpressions()
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

func TestTrack(t *testing.T) {
	result := make(map[string][]*aggregatedRecord)
	cfg := MakeTrackerConfig("test", "test",
		testLabelGetters,
		makeTestGatherFunc(testData),
		func(metricName string, labels prometheus.Labels, total int) {
			result[metricName] = append(result[metricName], &aggregatedRecord{labels, total})
		},
	)
	cfg.metricsConfig = makeTestMetricLabelExpressions(t)
	cfg.Track(context.Background())

	if assert.Len(t, result, 2) &&
		assert.Contains(t, result, "TestTrack_metric1") &&
		assert.Contains(t, result, "TestTrack_metric2") {

		assert.ElementsMatch(t, result["TestTrack_metric1"],
			[]*aggregatedRecord{
				{prometheus.Labels{
					"Severity": "CRITICAL",
					"Cluster":  "cluster 1",
				}, 2},
				{prometheus.Labels{
					"Severity": "HIGH",
					"Cluster":  "cluster 2",
				}, 1},
			})

		assert.ElementsMatch(t, result["TestTrack_metric2"],
			[]*aggregatedRecord{
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

func TestTrackerConfig_registerMetrics(t *testing.T) {
	tc := MakeTrackerConfig("test", "test",
		testLabelGetters, nil, nil)
	testRegistry := prometheus.NewRegistry()
	tc.metricsConfig = makeTestMetricLabelExpressions(t)
	tc.metricsConfig["m1"] = map[Label][]*Expression{
		"l1": nil,
	}
	assert.NoError(t, tc.registerMetrics(testRegistry, time.Hour))
	assert.NoError(t, tc.registerMetrics(testRegistry, time.Hour))
	tc.metricsConfig["m1"] = map[Label][]*Expression{
		"l1": nil,
		"l2": nil,
	}
	assert.Error(t, tc.registerMetrics(testRegistry, time.Hour))
	tc.metricsConfig["m1"] = map[Label][]*Expression{
		"l2": nil,
	}
	assert.Error(t, tc.registerMetrics(testRegistry, time.Hour))
	tc.metricsConfig["m1"] = map[Label][]*Expression{
		"l1": nil,
	}
	assert.NoError(t, tc.registerMetrics(testRegistry, time.Hour))
}
