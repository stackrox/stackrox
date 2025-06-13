package common

import (
	"context"
	"iter"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func nilGatherFunc(context.Context, *v1.Query, MetricsConfiguration) iter.Seq[OneOrMore] {
	return func(yield func(OneOrMore) bool) {}
}

func makeTestGatherFunc(data []map[Label]string) FindingGenerator[OneOrMore] {
	return func(context.Context, *v1.Query, MetricsConfiguration) iter.Seq[OneOrMore] {
		return func(yield func(OneOrMore) bool) {
			for i := range data {
				if !yield(OneOrMore(i)) {
					return
				}
			}
		}
	}
}

func TestMakeTrackerBase(t *testing.T) {
	tracker := MakeTrackerBase("test", "test", testLabelGetters, nilGatherFunc, nil)
	assert.NotNil(t, tracker)
	assert.Nil(t, tracker.ticker)

	cfg := tracker.GetConfiguration()
	if assert.NotNil(t, cfg) {
		assert.Empty(t, cfg.filter)
		assert.Nil(t, cfg.metrics)
		assert.Nil(t, cfg.metricRegistry)
		assert.Empty(t, cfg.toAdd)
		assert.Empty(t, cfg.toDelete)
		assert.Zero(t, cfg.period)
	}
}

func TestTrackerBase_Reconfigure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("test 0 period", func(t *testing.T) {
		tracker := MakeTrackerBase("test", "test", testLabelGetters, nilGatherFunc, nil)
		cfg0 := &Configuration{
			filter: &v1.Query{},
		}
		calledRegistry := false
		tracker.registerMetricFunc = func(*Configuration, MetricName) { calledRegistry = true }
		tracker.unregisterMetricFunc = func(MetricName) { calledRegistry = true }

		tracker.Reconfigure(ctx, cfg0)
		config := tracker.GetConfiguration()
		assert.Same(t, cfg0, config)
		assert.Nil(t, tracker.ticker)

		cfg1 := &Configuration{
			filter: &v1.Query{},
		}
		tracker.Reconfigure(ctx, cfg1)
		assert.Same(t, cfg1, tracker.GetConfiguration())
		assert.False(t, calledRegistry)
	})

	t.Run("test add -> delete -> stop", func(t *testing.T) {
		result := make(map[MetricName][]*aggregatedRecord)
		tracker := MakeTrackerBase("test", "test", testLabelGetters,
			makeTestGatherFunc(testData),
			func(metricName string, labels prometheus.Labels, total int) {
				result[MetricName(metricName)] = append(result[MetricName(metricName)],
					&aggregatedRecord{labels, total})
			})
		var registered, unregistered []MetricName
		tracker.registerMetricFunc = func(_ *Configuration, metric MetricName) { registered = append(registered, metric) }
		tracker.unregisterMetricFunc = func(metric MetricName) { unregistered = append(unregistered, metric) }

		mcfg := makeTestMetricLabelExpression(t)
		metricNames := slices.Collect(maps.Keys(mcfg))
		// Add test metrics:
		cfg0 := &Configuration{
			metrics: mcfg,
			toAdd:   metricNames,
			period:  time.Hour,
		}
		tracker.Reconfigure(ctx, cfg0)
		config := tracker.GetConfiguration()
		assert.Same(t, cfg0, config)
		assert.NotNil(t, tracker.ticker)
		assert.ElementsMatch(t, cfg0.toAdd, registered)
		assert.Empty(t, unregistered)
		// track() is called, so there is result from all metrics:
		assert.ElementsMatch(t, slices.Collect(maps.Keys(result)), metricNames)

		// Delete one random metric and update ticker:
		result = make(map[MetricName][]*aggregatedRecord)
		registered = []MetricName{}
		unregistered = []MetricName{}
		delete(mcfg, metricNames[0])
		cfg1 := &Configuration{
			metrics:  mcfg,
			toDelete: metricNames[:1],
			period:   2 * time.Hour,
		}
		tracker.Reconfigure(ctx, cfg1)
		assert.Same(t, cfg1, tracker.GetConfiguration())
		assert.Empty(t, registered)
		assert.ElementsMatch(t, cfg1.toDelete, unregistered)
		// track() is called, so some result should be gathered from the
		// persisted metric:
		assert.ElementsMatch(t, slices.Collect(maps.Keys(result)), metricNames[1:])

		// Stop and unregister everything:
		result = make(map[MetricName][]*aggregatedRecord)
		registered = []MetricName{}
		unregistered = []MetricName{}
		cfg2 := &Configuration{
			metrics: mcfg,
			period:  0,
		}
		tracker.Reconfigure(ctx, cfg2)
		assert.Same(t, cfg2, tracker.GetConfiguration())
		assert.Empty(t, registered)
		assert.ElementsMatch(t, metricNames[1:], unregistered)
		assert.Empty(t, result)
	})

}

func TestTrackerBase_Track(t *testing.T) {
	result := make(map[string][]*aggregatedRecord)
	tracker := MakeTrackerBase("test", "test",
		testLabelGetters,
		makeTestGatherFunc(testData),
		func(metricName string, labels prometheus.Labels, total int) {
			result[metricName] = append(result[metricName], &aggregatedRecord{labels, total})
		},
	)
	tracker.config.metrics = makeTestMetricLabelExpression(t)
	tracker.track(context.Background())

	if assert.Len(t, result, 2) &&
		assert.Contains(t, result, "TestTrackerBase_Track_metric1") &&
		assert.Contains(t, result, "TestTrackerBase_Track_metric2") {

		assert.ElementsMatch(t, result["TestTrackerBase_Track_metric1"],
			[]*aggregatedRecord{
				{prometheus.Labels{
					"Severity": "CRITICAL",
					"Cluster":  "cluster 1",
				}, 4},
			})

		assert.ElementsMatch(t, result["TestTrackerBase_Track_metric2"],
			[]*aggregatedRecord{
				{prometheus.Labels{
					"Namespace": "ns 1",
				}, 1},
				{prometheus.Labels{
					"Namespace": "ns 2",
				}, 1},
				{prometheus.Labels{
					"Namespace": "ns 3",
				}, 9},
			})
	}
}

func TestTrackerBase_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	result := make(map[string][]*aggregatedRecord)
	tracker := MakeTrackerBase("test", "test",
		testLabelGetters,
		makeTestGatherFunc(testData),
		func(metricName string, labels prometheus.Labels, total int) {
			result[metricName] = append(result[metricName], &aggregatedRecord{labels, total})
			cancel()
		},
	)
	mcfg := makeTestMetricLabelExpression(t)
	tracker.Reconfigure(ctx, &Configuration{
		metrics: mcfg,
		toAdd:   slices.Collect(maps.Keys(mcfg)),
		period:  time.Hour,
	})
	// The gauge function cancels the context, so Run should not hang.
	tracker.Run(ctx)
	assert.NotEmpty(t, result)
}
