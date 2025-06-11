package common

import (
	"context"
	"iter"
	"strings"
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
	assert.Nil(t, tracker.ticker)

	query, mcfg := tracker.GetMetricsConfiguration()
	assert.Empty(t, query)
	assert.Nil(t, mcfg)
}

func TestTrackerConfig_Reconfigure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("test 0 period", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(ctx, "", nil, 0))
		query, mcfg := tracker.GetMetricsConfiguration()
		assert.Nil(t, query)
		assert.Nil(t, mcfg)
	})

	t.Run("test query", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(ctx, `Cluster:name`, makeTestMetricLabels(t), 0))
		query, mcfg := tracker.GetMetricsConfiguration()
		assert.Equal(t, "Cluster", query.GetBaseQuery().GetMatchFieldQuery().GetField())
		assert.NotNil(t, mcfg)
	})

	t.Run("test bad query", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)

		assert.NoError(t, tracker.Reconfigure(ctx, `bad query?`, nil, 0))
	})

	t.Run("test with good test configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(ctx, "", makeTestMetricLabels(t), 42*time.Hour))
		_, mcfg := tracker.GetMetricsConfiguration()
		assert.NotNil(t, mcfg)
		assert.Equal(t, makeTestMetricLabelExpression(t), mcfg)
	})

	t.Run("test with initial bad configuration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		err := tracker.Reconfigure(ctx, "", map[string]*storage.PrometheusMetricsConfig_Labels{
			" ": nil,
		}, 11*time.Hour)

		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Equal(t, `invalid configuration: invalid metric name " ": doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`, err.Error())

		_, mcfg := tracker.GetMetricsConfiguration()
		assert.Nil(t, mcfg)
	})

	t.Run("test with bad reconfiguration", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(ctx, "", makeTestMetricLabels(t), 42*time.Hour))

		err := tracker.Reconfigure(ctx, "", map[string]*storage.PrometheusMetricsConfig_Labels{
			"m1": {
				Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
					"label1": nil,
				},
			},
		}, 11*time.Hour)
		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Equal(t, `invalid configuration: label "label1" for metric "m1" is not in the list of known labels: [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())

		_, mcfg := tracker.GetMetricsConfiguration()
		assert.NotNil(t, mcfg)
		assert.Equal(t, makeTestMetricLabelExpression(t), mcfg)
	})

	t.Run("change exposure", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(ctx, "", makeTestMetricLabels(t), 42*time.Hour))
		cfg := makeTestMetricLabels(t)
		for _, config := range cfg {
			if config.Exposure == storage.PrometheusMetricsConfig_Labels_BOTH {
				config.Exposure = storage.PrometheusMetricsConfig_Labels_INTERNAL
			}
		}
		assert.ErrorIs(t, tracker.Reconfigure(ctx, "", cfg, 42*time.Hour), errInvalidConfiguration)
	})

	t.Run("change labels", func(t *testing.T) {
		tracker := MakeTrackerConfig("test", "test", testLabelGetters, nilGatherFunc, nil)
		assert.NoError(t, tracker.Reconfigure(ctx, "", makeTestMetricLabels(t), 42*time.Hour))
		cfg := makeTestMetricLabels(t)
		for _, config := range cfg {
			if config.Exposure == storage.PrometheusMetricsConfig_Labels_BOTH {
				config.Labels["CVE"] = &storage.PrometheusMetricsConfig_Labels_Expression{}
			}
		}
		err := tracker.Reconfigure(ctx, "", cfg, 42*time.Hour)
		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.True(t, strings.Contains(err.Error(), "cannot alter metrics"))
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
	cfg.track(context.Background())

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
