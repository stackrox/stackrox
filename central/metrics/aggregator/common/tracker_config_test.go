package common

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

type testDataIndex int

func (t testDataIndex) Count() int {
	return 1
}

var testRegistry = prometheus.NewRegistry()

func nilGatherFunc(context.Context, *v1.Query, MetricsConfiguration) iter.Seq[testDataIndex] {
	return func(yield func(testDataIndex) bool) {}
}

func makeTestGatherFunc(data []map[Label]string) FindingGenerator[testDataIndex] {
	return func(context.Context, *v1.Query, MetricsConfiguration) iter.Seq[testDataIndex] {
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

	query, mcfg := tracker.GetMetricsConfiguration()
	assert.Empty(t, query)
	assert.Nil(t, mcfg)
}

func TestTrackerConfig_Reconfigure(t *testing.T) {

	t.Run("test 0 period", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(testRegistry, "", nil, 0))
		query, mcfg := tracker.GetMetricsConfiguration()
		assert.Equal(t, "", query.String())
		assert.Nil(t, mcfg)
	})

	t.Run("test query", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(testRegistry, `Cluster:name`, nil, 0))
		query, mcfg := tracker.GetMetricsConfiguration()
		assert.Equal(t, "Cluster", query.GetBaseQuery().GetMatchFieldQuery().GetField())
		assert.Nil(t, mcfg)
	})

	t.Run("test bad query", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(testRegistry, `bad query?`, nil, 0))
	})

	t.Run("test with good test configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(testRegistry, "", makeTestMetricLabels(t), 42*time.Hour))
		_, mcfg := tracker.GetMetricsConfiguration()
		assert.NotNil(t, mcfg)
		select {
		case period := <-tracker.GetPeriodCh():
			assert.Equal(t, 42*time.Hour, period)
		default:
			assert.Fail(t, "should have period configured")
		}
		assert.Equal(t, makeTestMetricLabelExpression(t), mcfg)
	})

	t.Run("test with initial bad configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		err := tracker.Reconfigure(testRegistry, "", map[string]*storage.PrometheusMetricsConfig_Labels{
			" ": nil,
		}, 11*time.Hour)

		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Equal(t, `invalid configuration: invalid metric name " ": doesn't match "^[a-zA-Z0-9_]+$"`, err.Error())

		_, mcfg := tracker.GetMetricsConfiguration()
		assert.Nil(t, mcfg)
		select {
		case period := <-tracker.GetPeriodCh():
			assert.Fail(t, "period configured: %v", period)
		default:
		}
	})

	t.Run("test with bad reconfiguration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(testRegistry, "", makeTestMetricLabels(t), 42*time.Hour))

		err := tracker.Reconfigure(testRegistry, "", map[string]*storage.PrometheusMetricsConfig_Labels{
			"m1": {
				Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
					"label1": nil,
				},
			},
		}, 11*time.Hour)
		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Equal(t, `invalid configuration: label "label1" for metric "m1" is not in the list of known labels: [test Cluster Namespace CVE Severity CVSS IsFixable]`, err.Error())

		_, mcfg := tracker.GetMetricsConfiguration()
		assert.NotNil(t, mcfg)
		select {
		case period := <-tracker.GetPeriodCh():
			assert.Equal(t, 42*time.Hour, period)
		default:
			assert.Fail(t, "no period in the channel")
		}
		assert.Equal(t, makeTestMetricLabelExpression(t), mcfg)
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
	cfg.metricsConfig = makeTestMetricLabelExpression(t)
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
	tc.metricsConfig = makeTestMetricLabelExpression(t)
	tc.metricsConfig["m1"] = map[Label]Expression{
		"l1": nil,
	}
	assert.NoError(t, tc.registerMetrics(testRegistry, time.Hour))
	assert.NoError(t, tc.registerMetrics(testRegistry, time.Hour))
	tc.metricsConfig["m1"] = map[Label]Expression{
		"l1": nil,
		"l2": nil,
	}
	assert.Error(t, tc.registerMetrics(testRegistry, time.Hour))
	tc.metricsConfig["m1"] = map[Label]Expression{
		"l2": nil,
	}
	assert.Error(t, tc.registerMetrics(testRegistry, time.Hour))
	tc.metricsConfig["m1"] = map[Label]Expression{
		"l1": nil,
	}
	assert.NoError(t, tc.registerMetrics(testRegistry, time.Hour))
}
