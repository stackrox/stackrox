package tracker

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/mocks"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
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
	assert.True(t, tracker.scoped)
}

func TestMakeTrackerBase_PanicsOnScopedCounter(t *testing.T) {
	assert.Panics(t, func() {
		MakeTrackerBase("test", "Test", testLabelGetters, nil)
	}, "should panic when creating scoped counter tracker")
}

func Test_makeTrackerBase(t *testing.T) {
	// Test global counter tracker.
	tracker := makeTrackerBase("test", "Test", false, testLabelGetters, nil)
	assert.NotNil(t, tracker)
	assert.Nil(t, tracker.getConfiguration())
	assert.False(t, tracker.scoped)

	tracker.Reconfigure(&Configuration{
		metrics: makeTestMetricDescriptors(t),
		enabled: true,
	})

	// After Reconfigure, no gatherer should exist yet (lazy creation).
	i := 0
	tracker.gatherers.Range(func(key, value any) bool {
		i++
		return true
	})
	assert.Equal(t, 0, i, "no gatherers should exist after Reconfigure")

	// Simulate first increment to trigger lazy creation.
	tracker.IncrementCounter(testFinding(0))

	// Now the global gatherer should exist with running=true.
	i = 0
	var id string
	var g *gatherer[testFinding]
	tracker.gatherers.Range(func(key, value any) bool {
		i++
		id = key.(string)
		g = value.(*gatherer[testFinding])
		return true
	})
	assert.Equal(t, 1, i)
	assert.Equal(t, globalScopeID, id)
	if assert.NotNil(t, g) {
		assert.True(t, g.running.Load(), "gatherer should be marked as running to prevent cleanup")
	}
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
		rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any()).
			AnyTimes().Do(
			func(metricName string, _ string, _ []string) {
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
			g := gRaw.(*gatherer[testFinding])
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
	testGatherer := &gatherer[testFinding]{
		registry:   rf,
		aggregator: makeAggregator(tracker.config.metrics, tracker.config.includeFilters, tracker.config.excludeFilters, tracker.getters),
	}
	assert.NoError(t, tracker.track(context.Background(), testGatherer, tracker.config))

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
	testGatherer := &gatherer[testFinding]{
		registry:   rf,
		aggregator: makeAggregator(tracker.config.metrics, tracker.config.includeFilters, tracker.config.excludeFilters, tracker.getters),
	}
	assert.ErrorIs(t, tracker.track(context.Background(), testGatherer, tracker.config),
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
		rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any()).
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

func TestTrackerBase_Gather_resetBetweenRuns(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)
	tracker := MakeTrackerBase("test", "Test",
		testLabelGetters,
		makeTestGatherFunc(testData),
	)
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

	// Track all SetTotal calls with their totals.
	var allTotals []int
	rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
	rf.EXPECT().SetTotal(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().Do(func(_ string, _ prometheus.Labels, total int) {
		allTotals = append(allTotals, total)
	})
	rf.EXPECT().Reset(gomock.Any()).AnyTimes()
	rf.EXPECT().Lock().AnyTimes()
	rf.EXPECT().Unlock().AnyTimes()

	md := makeTestMetricDescriptors(t)
	tracker.Reconfigure(&Configuration{
		metrics: md,
		toAdd:   slices.Collect(maps.Keys(md)),
		period:  time.Nanosecond, // Very short period to allow immediate re-gather.
	})

	ctx := makeAdminContext(t)

	// First gather.
	tracker.Gather(ctx)
	tracker.cleanupWG.Wait()
	firstRunTotals := append([]int{}, allTotals...)
	assert.NotEmpty(t, firstRunTotals, "first gather should produce results")

	// Reset tracking and force immediate re-gather.
	allTotals = nil
	identity, _ := authn.IdentityFromContext(ctx)
	gRaw, _ := tracker.gatherers.Load(identity.UID())
	g := gRaw.(*gatherer[testFinding])
	require.Eventually(t, g.trySetRunning, 5*time.Second, 10*time.Millisecond)
	g.lastGather = time.Time{} // Reset to allow immediate gather.
	g.running.Store(false)

	// Second gather.
	tracker.Gather(ctx)
	tracker.cleanupWG.Wait()

	// Verify second run produced the same totals (not accumulated).
	assert.ElementsMatch(t, firstRunTotals, allTotals,
		"second gather should produce same totals, not accumulated values")
}

func TestTrackerBase_Gather_afterReconfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	rf := mocks.NewMockCustomRegistry(ctrl)
	tracker := MakeTrackerBase("test", "Test",
		testLabelGetters,
		makeTestGatherFunc(testData),
	)
	tracker.registryFactory = func(string) (metrics.CustomRegistry, error) { return rf, nil }

	var gatheredMetrics []string
	rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	rf.EXPECT().UnregisterMetric(gomock.Any()).AnyTimes()
	rf.EXPECT().SetTotal(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().Do(func(metricName string, _ prometheus.Labels, _ int) {
		gatheredMetrics = append(gatheredMetrics, metricName)
	})
	rf.EXPECT().Reset(gomock.Any()).AnyTimes()
	rf.EXPECT().Lock().AnyTimes()
	rf.EXPECT().Unlock().AnyTimes()

	// Initial configuration with 2 metrics.
	md1 := makeTestMetricDescriptors(t)
	cfg1 := &Configuration{
		metrics: md1,
		toAdd:   slices.Collect(maps.Keys(md1)),
		period:  time.Nanosecond,
	}
	tracker.Reconfigure(cfg1)

	ctx := makeAdminContext(t)
	tracker.Gather(ctx)
	tracker.cleanupWG.Wait()

	assert.NotEmpty(t, gatheredMetrics, "should gather metrics with initial config")
	initialMetrics := slices.Compact(slices.Sorted(slices.Values(gatheredMetrics)))

	// Reconfigure with different metrics (only metric1).
	gatheredMetrics = nil
	md2 := MetricDescriptors{
		"test_" + MetricName(t.Name()) + "_new_metric": {"Severity"},
	}
	cfg2 := &Configuration{
		metrics:  md2,
		toAdd:    []MetricName{"test_" + MetricName(t.Name()) + "_new_metric"},
		toDelete: slices.Collect(maps.Keys(md1)),
		period:   time.Nanosecond,
	}
	tracker.Reconfigure(cfg2)

	// Reset lastGather to allow immediate gather.
	identity, _ := authn.IdentityFromContext(ctx)
	gRaw, _ := tracker.gatherers.Load(identity.UID())
	g := gRaw.(*gatherer[testFinding])
	require.Eventually(t, g.trySetRunning, 5*time.Second, 10*time.Millisecond)
	g.lastGather = time.Time{}
	g.running.Store(false)

	// Gather with new configuration.
	tracker.Gather(ctx)
	tracker.cleanupWG.Wait()

	assert.NotEmpty(t, gatheredMetrics, "should gather metrics after reconfiguration")
	newMetrics := slices.Compact(slices.Sorted(slices.Values(gatheredMetrics)))

	// Verify we're gathering different metrics now.
	assert.NotEqual(t, initialMetrics, newMetrics,
		"metrics after reconfiguration should differ from initial")
	assert.Contains(t, newMetrics, "test_"+t.Name()+"_new_metric",
		"should contain the new metric")
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
	rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any()).
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
	rf.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any()).
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
	props := tracker.makeProps(descriptionTitle)
	get := func(key string) any {
		if v, ok := props[key]; ok {
			return v
		}
		return nil
	}

	assert.Len(t, props, 2)
	assert.ElementsMatch(t, get("Telemetry test metrics labels"), []Label{"Cluster", "Namespace", "Severity"})
	assert.Equal(t, len(md), get("Total Telemetry test metrics"))
}

func Test_formatMetricsHelp(t *testing.T) {
	assert.Equal(t, "The total number of my metric",
		formatMetricHelp("my metric", &Configuration{}, "my_metric"))

	assert.Equal(t, `The total number of my metric aggregated by Label1, Label2, and gathered every 1h0m0s`,
		formatMetricHelp("my metric", &Configuration{
			metrics: MetricDescriptors{
				"metric1": []Label{"Label1", "Label2"},
			},
			period: time.Hour,
		}, "metric1"))

	assert.Equal(t, `The total number of my metric aggregated by Label1, Label2, including only Label3≈"EXPR1", Label4≈"EXPR11", excluding Label4≈"EXPR2", Label5≈"EXPR2", and gathered every 1h0m0s`,
		formatMetricHelp("my metric", &Configuration{
			metrics: MetricDescriptors{
				"metric1": []Label{"Label1", "Label2"},
			},
			includeFilters: LabelFilters{
				"metric1": {
					"Label3": regexp.MustCompile("EXPR1"),
					"Label4": regexp.MustCompile("EXPR11"),
				},
			},
			excludeFilters: LabelFilters{
				"metric1": {
					"Label4": regexp.MustCompile("EXPR2"),
					"Label5": regexp.MustCompile("EXPR2"),
				},
			},
			period: time.Hour,
		}, "metric1"))
}

func makeScopedGatherFunc(ctx context.Context, _ MetricDescriptors) FindingErrorSequence[testFinding] {
	return func(yield func(testFinding, error) bool) {
		identity, err := authn.IdentityFromContext(ctx)
		if err != nil {
			return
		}

		username := identity.UID()
		var finding testFinding
		for idx := range testData {
			cluster := testData[idx]["Cluster"]

			shouldYield := false
			switch username {
			case accesscontrol.Admin:
				shouldYield = true
			case "No Access": // see basic.ContextWithNoAccessIdentity code.
				shouldYield = cluster == "cluster 1"
			case accesscontrol.None:
				shouldYield = false
			}

			if shouldYield && !yield(finding, nil) {
				return
			}
			finding++
		}
	}
}

func makeTestCtxIdentity(t *testing.T, provider authproviders.Provider, accessCtxFunc func(*testing.T, authproviders.Provider) context.Context) (context.Context, authn.Identity) {
	ctx := accessCtxFunc(t, provider)
	id, _ := authn.IdentityFromContext(ctx)
	return ctx, id
}

func Test_scope(t *testing.T) {
	t.Run("scoped access", func(t *testing.T) {
		tracker := MakeTrackerBase("test", "Test",
			testLabelGetters,
			makeScopedGatherFunc)

		md := makeTestMetricDescriptors(t)
		tracker.Reconfigure(&Configuration{
			metrics: md,
			toAdd:   slices.Collect(maps.Keys(md)),
			period:  time.Hour,
		})

		provider, _ := authproviders.NewProvider(
			authproviders.WithEnabled(true),
			authproviders.WithID(uuid.NewV4().String()),
			authproviders.WithName("test"),
		)

		adminCtx, adminIdentity := makeTestCtxIdentity(t, provider,
			basic.ContextWithAdminIdentity)
		cluster1Ctx, cluster1Identity := makeTestCtxIdentity(t, provider,
			basic.ContextWithNoAccessIdentity)
		noAccessCtx, noAccessIdentity := makeTestCtxIdentity(t, provider,
			basic.ContextWithNoneIdentity)

		tracker.Gather(adminCtx)
		tracker.Gather(cluster1Ctx)
		tracker.Gather(noAccessCtx)
		tracker.cleanupWG.Wait()

		adminGatherer, _ := tracker.gatherers.Load(adminIdentity.UID())
		cluster1Gatherer, _ := tracker.gatherers.Load(cluster1Identity.UID())
		noAccessGatherer, _ := tracker.gatherers.Load(noAccessIdentity.UID())

		adminRegistry := adminGatherer.(*gatherer[testFinding]).registry
		cluster1Registry := cluster1Gatherer.(*gatherer[testFinding]).registry
		noAccessRegistry := noAccessGatherer.(*gatherer[testFinding]).registry

		adminMetrics := readMetrics(adminRegistry)
		cluster1Metrics := readMetrics(cluster1Registry)
		noAccessMetrics := readMetrics(noAccessRegistry)

		const expectedAdminMetrics = `# HELP rox_central_test_Test_scope_scoped_access_metric1 The total number of Test aggregated by Cluster,Severity and gathered every 1h0m0s
# TYPE rox_central_test_Test_scope_scoped_access_metric1 gauge
rox_central_test_Test_scope_scoped_access_metric1{Cluster="cluster 1",Severity="CRITICAL"} 2
rox_central_test_Test_scope_scoped_access_metric1{Cluster="cluster 2",Severity="HIGH"} 1
rox_central_test_Test_scope_scoped_access_metric1{Cluster="cluster 3",Severity="LOW"} 1
rox_central_test_Test_scope_scoped_access_metric1{Cluster="cluster 5",Severity="LOW"} 1
# HELP rox_central_test_Test_scope_scoped_access_metric2 The total number of Test aggregated by Namespace and gathered every 1h0m0s
# TYPE rox_central_test_Test_scope_scoped_access_metric2 gauge
rox_central_test_Test_scope_scoped_access_metric2{Namespace="ns 1"} 1
rox_central_test_Test_scope_scoped_access_metric2{Namespace="ns 2"} 1
rox_central_test_Test_scope_scoped_access_metric2{Namespace="ns 3"} 3
`

		const expectedCluster1Metrics = `# HELP rox_central_test_Test_scope_scoped_access_metric1 The total number of Test aggregated by Cluster,Severity and gathered every 1h0m0s
# TYPE rox_central_test_Test_scope_scoped_access_metric1 gauge
rox_central_test_Test_scope_scoped_access_metric1{Cluster="cluster 1",Severity="CRITICAL"} 2
# HELP rox_central_test_Test_scope_scoped_access_metric2 The total number of Test aggregated by Namespace and gathered every 1h0m0s
# TYPE rox_central_test_Test_scope_scoped_access_metric2 gauge
rox_central_test_Test_scope_scoped_access_metric2{Namespace="ns 1"} 1
rox_central_test_Test_scope_scoped_access_metric2{Namespace="ns 3"} 1
`

		assert.Equal(t, expectedAdminMetrics, adminMetrics)
		assert.Equal(t, expectedCluster1Metrics, cluster1Metrics)
		assert.Empty(t, noAccessMetrics)

		t.Cleanup(func() {
			metrics.DeleteCustomRegistry(adminIdentity.UID())
			metrics.DeleteCustomRegistry(cluster1Identity.UID())
			metrics.DeleteCustomRegistry(noAccessIdentity.UID())
		})
	})

	t.Run("global access", func(t *testing.T) {
		tracker := MakeGlobalTrackerBase("test", "Test",
			testLabelGetters,
			makeTestGatherFunc(testData))

		md := makeTestMetricDescriptors(t)
		tracker.Reconfigure(&Configuration{
			metrics: md,
			toAdd:   slices.Collect(maps.Keys(md)),
			period:  time.Hour,
		})

		provider, _ := authproviders.NewProvider(
			authproviders.WithEnabled(true),
			authproviders.WithID(uuid.NewV4().String()),
			authproviders.WithName("test"),
		)

		adminCtx, _ := makeTestCtxIdentity(t, provider,
			basic.ContextWithAdminIdentity)
		cluster1Ctx, _ := makeTestCtxIdentity(t, provider,
			basic.ContextWithNoAccessIdentity)
		noAccessCtx, _ := makeTestCtxIdentity(t, provider,
			basic.ContextWithNoneIdentity)

		tracker.Gather(adminCtx)
		tracker.Gather(cluster1Ctx)
		tracker.Gather(noAccessCtx)
		tracker.cleanupWG.Wait()

		globalRegistry, err := metrics.GetGlobalRegistry()
		require.NoError(t, err)

		const expectedMetrics = `# HELP rox_central_test_Test_scope_global_access_metric1 The total number of Test aggregated by Cluster,Severity and gathered every 1h0m0s
# TYPE rox_central_test_Test_scope_global_access_metric1 gauge
rox_central_test_Test_scope_global_access_metric1{Cluster="cluster 1",Severity="CRITICAL"} 2
rox_central_test_Test_scope_global_access_metric1{Cluster="cluster 2",Severity="HIGH"} 1
rox_central_test_Test_scope_global_access_metric1{Cluster="cluster 3",Severity="LOW"} 1
rox_central_test_Test_scope_global_access_metric1{Cluster="cluster 5",Severity="LOW"} 1
# HELP rox_central_test_Test_scope_global_access_metric2 The total number of Test aggregated by Namespace and gathered every 1h0m0s
# TYPE rox_central_test_Test_scope_global_access_metric2 gauge
rox_central_test_Test_scope_global_access_metric2{Namespace="ns 1"} 1
rox_central_test_Test_scope_global_access_metric2{Namespace="ns 2"} 1
rox_central_test_Test_scope_global_access_metric2{Namespace="ns 3"} 3
`

		// All users should see the same global metrics.
		globalMetrics := readMetrics(globalRegistry)
		assert.Equal(t, expectedMetrics, globalMetrics)
	})
}

func readMetrics(registry metrics.CustomRegistry) string {
	rec := httptest.NewRecorder()
	registry.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rec.Body.String()
}

func TestTrackerBase_IncrementCounter(t *testing.T) {
	t.Run("increments counter in global registry", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		// Create a global counter tracker (nil generator)
		tracker := MakeGlobalTrackerBase("test", "test counter", testLabelGetters, nil)

		// Create mock global registry
		mockRegistry := mocks.NewMockCustomRegistry(ctrl)

		tracker.registryFactory = func(userID string) (metrics.CustomRegistry, error) {
			if userID == globalScopeID {
				return mockRegistry, nil
			}
			return nil, errox.InvariantViolation.Newf("unexpected userID: %s", userID)
		}

		// Configure the tracker
		md := MetricDescriptors{
			"test_counter": {"Cluster", "Severity"},
		}
		cfg := &Configuration{
			metrics: md,
			toAdd:   slices.Collect(maps.Keys(md)),
			period:  0, // Counter trackers don't use period
			enabled: true,
		}

		// Expect RegisterCounter to be called when the global gatherer is created
		mockRegistry.EXPECT().RegisterCounter("test_counter", "test counter", []string{"Cluster", "Severity"}).Return(nil)

		tracker.Reconfigure(cfg)

		// Expect RegisterCounter to be called for each registry when gatherers are created
		mockRegistry.EXPECT().RegisterCounter("test_counter", "test counter", []string{"Cluster", "Severity"}).Return(nil)

		// Create gatherers directly by accessing the internal getGatherer method for testing
		// In production, gatherers are created when Gather() is called
		tracker.getGatherer("user1", cfg)
		tracker.getGatherer("user2", cfg)
		tracker.getGatherer("user3", cfg)

		// Set gatherers to not running so they can be used
		tracker.gatherers.Range(func(_, v any) bool {
			v.(*gatherer[testFinding]).running.Store(false)
			return true
		})

		// Expect IncrementCounter to be called on ALL three registries
		// Expect IncrementCounter to be called on the global registry
		expectedLabels := prometheus.Labels{
			"Cluster":  "cluster 1",
			"Severity": "CRITICAL",
		}
		mockRegistry.EXPECT().IncrementCounter("test_counter", expectedLabels)

		// Increment the counter - this should lazy-create the global gatherer
		tracker.IncrementCounter(testFinding(0))

		// Verify the global gatherer exists and is marked as running
		gr, exists := tracker.gatherers.Load(globalScopeID)
		require.True(t, exists, "global gatherer should exist after first increment")
		require.True(t, gr.(*gatherer[testFinding]).running.Load(), "global gatherer should be marked as running")
	})

	t.Run("no-op when no gatherers exist", func(t *testing.T) {
		// Create a global counter tracker with no gatherers
		tracker := MakeGlobalTrackerBase("test", "test counter", testLabelGetters, nil)

		// Configure the tracker
		md := MetricDescriptors{
			"test_counter": {"Cluster"},
		}
		cfg := &Configuration{
			metrics: md,
			toAdd:   slices.Collect(maps.Keys(md)),
		}
		tracker.Reconfigure(cfg)

		// Increment should be a no-op (no panic, no error)
		tracker.IncrementCounter(testFinding(0))
	})

	t.Run("does nothing on gauge tracker", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		// Create a gauge tracker (non-nil generator)
		tracker := MakeTrackerBase("test", "test gauge", testLabelGetters, nilGatherFunc)

		mockRegistry := mocks.NewMockCustomRegistry(ctrl)
		tracker.registryFactory = func(string) (metrics.CustomRegistry, error) {
			return mockRegistry, nil
		}

		md := MetricDescriptors{
			"test_metric": {"Cluster"},
		}
		cfg := &Configuration{
			metrics: md,
			toAdd:   slices.Collect(maps.Keys(md)),
			period:  time.Hour,
		}
		tracker.Reconfigure(cfg)

		// Expect RegisterMetric (not RegisterCounter) to be called
		mockRegistry.EXPECT().RegisterMetric("test_metric", "test gauge", []string{"Cluster"}).Return(nil)

		tracker.getGatherer("user1", cfg)

		// IncrementCounter should be a no-op for gauge trackers
		// No IncrementCounter expectation - it should not be called
		tracker.IncrementCounter(testFinding(0))
	})

	t.Run("returns early when configuration is nil", func(t *testing.T) {
		tracker := MakeGlobalTrackerBase("test", "test counter", testLabelGetters, nil)

		// No configuration set
		// IncrementCounter should return early without panicking
		tracker.IncrementCounter(testFinding(0))
	})

	t.Run("extracts correct label values from finding", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		tracker := MakeGlobalTrackerBase("test", "test counter", testLabelGetters, nil)

		mockRegistry := mocks.NewMockCustomRegistry(ctrl)
		tracker.registryFactory = func(userID string) (metrics.CustomRegistry, error) {
			if userID == globalScopeID {
				return mockRegistry, nil
			}
			return nil, errox.InvariantViolation.Newf("unexpected userID: %s", userID)
		}

		md := MetricDescriptors{
			"test_counter": {"Cluster", "Namespace", "CVE", "Severity"},
		}
		cfg := &Configuration{
			metrics: md,
			toAdd:   slices.Collect(maps.Keys(md)),
			enabled: true,
		}

		// Expect RegisterCounter to be called when the global gatherer is lazy-created
		mockRegistry.EXPECT().RegisterCounter("test_counter", "test counter",
			[]string{"Cluster", "Namespace", "CVE", "Severity"}).Return(nil)

		tracker.getGatherer("user1", cfg)
		tracker.gatherers.Range(func(_, v any) bool {
			v.(*gatherer[testFinding]).running.Store(false)
			return true
		})
		tracker.Reconfigure(cfg)

		// Expect correct labels extracted from testFinding(1)
		expectedLabels := prometheus.Labels{
			"Cluster":   testData[1]["Cluster"],
			"Namespace": testData[1]["Namespace"],
			"CVE":       testData[1]["CVE"],
			"Severity":  testData[1]["Severity"],
		}
		mockRegistry.EXPECT().IncrementCounter("test_counter", expectedLabels)

		// This should lazy-create the global gatherer and increment the counter
		tracker.IncrementCounter(testFinding(1))
	})
}
