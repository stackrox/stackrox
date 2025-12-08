package tracker

import (
	"context"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func nilGatherFunc(context.Context, MetricDescriptors) FindingErrorSequence[testFinding] {
	return func(yield func(testFinding, error) bool) {}
}

func makeTestGatherFunc(data []map[Label]string) FindingGenerator[testFinding] {
	return func(context.Context, MetricDescriptors) FindingErrorSequence[testFinding] {
		return func(yield func(testFinding, error) bool) {
			var finding testFinding
			for range data {
				if !yield(finding, nil) {
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
		tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

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
		tracker.cleanupWG.Wait()
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
		tracker.cleanupWG.Wait()
		assert.Empty(t, trackedMetricNames)

		{ // Reset lastGather
			identity, _ := authn.IdentityFromContext(ctx)
			gRaw, ok := tracker.gatherers.Load(identity.UID())
			require.True(t, ok)
			g := gRaw.(*gatherer)
			// Make it temporarily running to avoid data race on lastGather.
			require.Eventually(t, g.trySetRunning, 5*time.Second, 10*time.Millisecond)
			g.lastGather = time.Time{}
			g.running.Store(false)
		}
		tracker.Gather(ctx)
		tracker.cleanupWG.Wait()

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
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

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
		func(context.Context, MetricDescriptors) FindingErrorSequence[testFinding] {
			return func(yield func(testFinding, error) bool) {
				if !yield(testFinding(0xbadf00d), errox.InvariantViolation.CausedBy("bad finding")) {
					return
				}
			}
		},
	)
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

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
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

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
	tracker.cleanupWG.Wait()
}

func TestTrackerBase_getGatherer(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)
	tracker := MakeTrackerBase("test", "test",
		testLabelGetters,
		makeTestGatherFunc(testData),
	)
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

	md := makeTestMetricDescriptors(t)
	tracker.Reconfigure(&Configuration{
		metrics: md,
		toAdd:   slices.Collect(maps.Keys(md)),
		period:  time.Hour,
	})

	cfg := tracker.getConfiguration()
	rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		// each new gatherer, created by getGatherer, calls RegisterMetric
		// for every metric.
		Times(2 * len(cfg.metrics))

	g := tracker.getGatherer("Admin", cfg)
	require.NotNil(t, g)
	tracker.cleanupWG.Wait()

	g.lastGather = time.Now().Add(-inactiveGathererTTL)
	g.running.Store(false)
	_, ok := tracker.gatherers.Load("Admin")
	assert.True(t, ok)
	tracker.getGatherer("Donkey", cfg).running.Store(false)
	// This call should delete the "Admin" gatherer:
	tracker.cleanupInactiveGatherers()
	tracker.cleanupWG.Wait()
	_, ok = tracker.gatherers.Load("Admin")
	assert.False(t, ok)
}

func TestTrackerBase_cleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)
	rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(4).Return(nil)

	// Test that cleanupInactiveGatherers removes gatherers that have been inactive for longer than inactiveGathererTTL.
	tracker := MakeTrackerBase("test", "test",
		testLabelGetters,
		makeTestGatherFunc(testData),
	)
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

	cfg := &Configuration{
		metrics: makeTestMetricDescriptors(t),
		toAdd:   []MetricName{"test_metric"},
		period:  time.Hour,
	}
	tracker.Reconfigure(cfg)

	// Add two gatherers: one active, one inactive.
	activeID := "active"
	inactiveID := "inactive"

	activeGatherer := tracker.getGatherer(activeID, cfg)
	inactiveGatherer := tracker.getGatherer(inactiveID, cfg)
	tracker.cleanupWG.Wait()

	// Set lastGather times.
	activeGatherer.lastGather = time.Now()
	activeGatherer.running.Store(false)
	inactiveGatherer.lastGather = time.Now().Add(-inactiveGathererTTL)
	inactiveGatherer.running.Store(false)

	// Sanity: both gatherers present.
	_, ok1 := tracker.gatherers.Load(activeID)
	_, ok2 := tracker.gatherers.Load(inactiveID)
	assert.True(t, ok1)
	assert.True(t, ok2)

	tracker.cleanupInactiveGatherers()
	tracker.cleanupWG.Wait()

	// Active gatherer should remain, inactive should be removed.
	_, ok1 = tracker.gatherers.Load(activeID)
	_, ok2 = tracker.gatherers.Load(inactiveID)
	assert.True(t, ok1, "active gatherer should not be removed")
	assert.False(t, ok2, "inactive gatherer should be removed")
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

	tracker := MakeTrackerBase("test", "telemetry test",
		testLabelGetters,
		makeTestGatherFunc(testData))
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

	md := makeTestMetricDescriptors(t)
	tracker.Reconfigure(&Configuration{
		metrics: md,
		toAdd:   slices.Collect(maps.Keys(md)),
		period:  time.Hour,
	})
	descriptionTitle := strings.ToTitle(tracker.description[0:1]) + tracker.description[1:]
	props := tracker.makeProps(descriptionTitle, 12345*time.Millisecond)
	get := func(key string) any {
		if v, ok := props[key]; ok {
			return v
		}
		return nil
	}

	assert.Len(t, props, 3)
	assert.ElementsMatch(t, get("Telemetry test metrics labels"), []Label{"Cluster", "Namespace", "Severity"})
	assert.Equal(t, len(md), get("Total Telemetry test metrics"))
	assert.Equal(t, uint32(12), get("Telemetry test gathering seconds"))
}
