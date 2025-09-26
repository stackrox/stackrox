package tracker

import (
	"context"
	"iter"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/mocks"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMakeLabelOrderMap(t *testing.T) {
	assert.Equal(t, map[Label]int{
		"test":      1,
		"Cluster":   2,
		"Namespace": 3,
		"CVE":       4,
		"Severity":  5,
		"CVSS":      6,
		"IsFixable": 7,
	}, testLabelOrder)
}

func nilGatherFunc(context.Context, MetricDescriptors) iter.Seq[testFinding] {
	return func(yield func(testFinding) bool) {}
}

func makeTestGatherFunc(data []map[Label]string) FindingGenerator[testFinding] {
	return func(context.Context, MetricDescriptors) iter.Seq[testFinding] {
		var finding testFinding
		return func(yield func(testFinding) bool) {
			for range data {
				if !yield(finding) {
					return
				}
				finding++
			}
		}
	}
}

func TestMakeTrackerBase(t *testing.T) {
	tracker := MakeTrackerBase("test", "Test", testLabelGetters, nilGatherFunc)
	assert.NotNil(t, tracker)
	assert.Nil(t, tracker.getConfiguration())
}

func TestTrackerBase_Reconfigure(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Run("nil configuration", func(t *testing.T) {
		tracker := MakeTrackerBase("test", "Test", testLabelGetters, nilGatherFunc)

		tracker.Reconfigure(nil)
		config := tracker.getConfiguration()
		if assert.NotNil(t, config) {
			assert.Nil(t, config.metrics)
			assert.Zero(t, config.period)
		}
	})

	t.Run("test 0 period", func(t *testing.T) {
		tracker := MakeTrackerBase("test", "Test", testLabelGetters, nilGatherFunc)
		cfg0 := &Configuration{}

		tracker.Reconfigure(cfg0)
		assert.Same(t, cfg0, tracker.getConfiguration())

		cfg1 := &Configuration{}
		tracker.Reconfigure(cfg1)
		assert.Same(t, cfg1, tracker.getConfiguration())
	})

	t.Run("test add -> delete -> stop", func(t *testing.T) {
		trackedMetricNames := make([]MetricName, 0)

		tracker := MakeTrackerBase("test", "Test", testLabelGetters,
			makeTestGatherFunc(testData))

		rf := mocks.NewMockCustomRegistry(ctrl)
		tracker.registryFactory = func(string) metrics.CustomRegistry { return rf }

		var registered, unregistered []MetricName
		rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes().Do(
			func(metricName string, _ string, _ time.Duration, _ []string) {
				registered = append(registered, MetricName(metricName))
			})
		rf.EXPECT().UnregisterMetric(gomock.Any()).
			AnyTimes().Do(
			func(metricName string) {
				unregistered = append(unregistered, MetricName(metricName))
			})
		rf.EXPECT().Lock().AnyTimes()
		rf.EXPECT().Unlock().AnyTimes()
		rf.EXPECT().Reset(gomock.Any()).AnyTimes()

		md := makeTestMetricDescriptors(t)
		metricNames := slices.Collect(maps.Keys(md))
		// Add test metrics:
		cfg0 := &Configuration{
			metrics: md,
			toAdd:   metricNames,
			period:  time.Hour,
		}
		// Initial configuration.
		tracker.Reconfigure(cfg0)
		config := tracker.getConfiguration()
		assert.Same(t, cfg0, config)
		assert.Empty(t, trackedMetricNames)

		ctx := makeAdminContext(t)
		rf.EXPECT().SetTotal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(
			func(metricName string, _ prometheus.Labels, _ int) {
				trackedMetricNames = append(trackedMetricNames, MetricName(metricName))
			},
		)

		tracker.Gather(ctx)
		assert.ElementsMatch(t, cfg0.toAdd, registered)
		assert.Empty(t, unregistered)
		assert.ElementsMatch(t, cfg0.toAdd, slices.Compact(trackedMetricNames))

		// Delete one random metric and update ticker:
		trackedMetricNames = nil
		registered = nil
		unregistered = nil
		delete(md, metricNames[0])
		cfg1 := &Configuration{
			metrics:  md,
			toDelete: metricNames[:1],
			period:   2 * time.Hour,
		}
		tracker.Reconfigure(cfg1)
		assert.Same(t, cfg1, tracker.getConfiguration())
		assert.Empty(t, registered)
		assert.ElementsMatch(t, cfg1.toDelete, unregistered)

		// Less than period since last Gather, gathering ignored:
		tracker.Gather(ctx)
		assert.Empty(t, trackedMetricNames)

		{ // Reset lastGather
			identity, _ := authn.IdentityFromContext(ctx)
			g, _ := tracker.gatherers.Load(identity.UID())
			g.(*gatherer).lastGather = time.Time{}
		}
		tracker.Gather(ctx)

		assert.ElementsMatch(t, slices.Compact(trackedMetricNames), metricNames[1:])

		// Stop and unregister everything:
		trackedMetricNames = nil
		registered = nil
		unregistered = nil
		tracker.Reconfigure(nil)

		assert.Empty(t, registered)
		assert.ElementsMatch(t, metricNames[1:], unregistered)
		assert.Empty(t, trackedMetricNames)
	})

}

func TestTrackerBase_Track(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)

	tracker := MakeTrackerBase("test", "Test",
		testLabelGetters,
		makeTestGatherFunc(testData))
	tracker.registryFactory = func(string) metrics.CustomRegistry { return rf }

	result := make(map[string][]*aggregatedRecord)
	rf.EXPECT().SetTotal(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().Do(
		func(metricName string, labels prometheus.Labels, total int) {
			result[metricName] = append(result[metricName], &aggregatedRecord{labels, total})
		},
	)

	rf.EXPECT().Reset("test_TestTrackerBase_Track_metric1").After(rf.EXPECT().Lock())
	rf.EXPECT().Unlock().After(rf.EXPECT().Reset("test_TestTrackerBase_Track_metric2"))

	tracker.config = &Configuration{
		metrics: makeTestMetricDescriptors(t),
	}
	assert.NoError(t, tracker.track(context.Background(), rf, tracker.config.metrics))

	if assert.Len(t, result, 2) &&
		assert.Contains(t, result, "test_TestTrackerBase_Track_metric1") &&
		assert.Contains(t, result, "test_TestTrackerBase_Track_metric2") {

		assert.ElementsMatch(t, result["test_TestTrackerBase_Track_metric1"],
			[]*aggregatedRecord{
				{prometheus.Labels{
					"Severity": "CRITICAL",
					"Cluster":  "cluster 1",
				}, 2},
				{prometheus.Labels{
					"Severity": "LOW",
					"Cluster":  "cluster 5",
				}, 1},
				{prometheus.Labels{
					"Severity": "LOW",
					"Cluster":  "cluster 3",
				}, 1},
				{prometheus.Labels{
					"Severity": "HIGH",
					"Cluster":  "cluster 2",
				}, 1},
			})

		assert.ElementsMatch(t, result["test_TestTrackerBase_Track_metric2"],
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

func TestTrackerBase_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)

	tracker := MakeTrackerBase("test", "Test",
		testLabelGetters,
		func(context.Context, MetricDescriptors) iter.Seq[testFinding] {
			return func(yield func(testFinding) bool) {
				if !yield(0xbadf00d) {
					return
				}
			}
		},
	)
	tracker.registryFactory = func(string) metrics.CustomRegistry { return rf }

	tracker.config = &Configuration{
		metrics: makeTestMetricDescriptors(t),
	}
	assert.ErrorIs(t, tracker.track(context.Background(), rf, tracker.config.metrics),
		errox.InvariantViolation)
}

func TestTrackerBase_Gather(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)
	tracker := MakeTrackerBase("test", "Test",
		testLabelGetters,
		makeTestGatherFunc(testData),
	)
	tracker.registryFactory = func(string) metrics.CustomRegistry { return rf }

	result := make(map[string][]*aggregatedRecord)
	{ // Capture result with a mock registry.
		rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(2)
		rf.EXPECT().SetTotal(gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes().Do(
			func(metricName string, labels prometheus.Labels, total int) {
				result[metricName] = append(result[metricName], &aggregatedRecord{labels, total})
			},
		)
		rf.EXPECT().Reset(gomock.Any()).AnyTimes()
		rf.EXPECT().Lock().AnyTimes()
		rf.EXPECT().Unlock().AnyTimes()
	}

	md := makeTestMetricDescriptors(t)
	tracker.Reconfigure(&Configuration{
		metrics: md,
		toAdd:   slices.Collect(maps.Keys(md)),
		period:  time.Hour,
	})

	ctx := makeAdminContext(t)
	tracker.Gather(ctx)
	assert.NotEmpty(t, result)
	result = make(map[string][]*aggregatedRecord)
	tracker.Gather(ctx)
	assert.Empty(t, result)
}

func makeAdminContext(t *testing.T) context.Context {
	authProvider, _ := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	ctx := basic.ContextWithAdminIdentity(t, authProvider)
	return ctx
}

func Test_makeProps(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)

	tracker := MakeTrackerBase("test", "test",
		testLabelGetters,
		makeTestGatherFunc(testData))
	tracker.registryFactory = func(string) metrics.CustomRegistry { return rf }

	md := makeTestMetricDescriptors(t)
	tracker.Reconfigure(&Configuration{
		metrics: md,
		toAdd:   slices.Collect(maps.Keys(md)),
		period:  time.Hour,
	})
	titCat := strings.ToTitle(tracker.description[0:1]) + tracker.description[1:]
	props := tracker.makeProps(titCat, 12345*time.Millisecond)
	get := func(key string) any {
		if v, ok := props[key]; ok {
			return v
		}
		return nil
	}

	assert.Len(t, props, 3)
	assert.ElementsMatch(t, get("Test metrics labels"), []Label{"Cluster", "Namespace", "Severity"})
	assert.Equal(t, len(md), get("Total Test metrics"))
	assert.Equal(t, uint32(12), get("Test gathering seconds"))
}
