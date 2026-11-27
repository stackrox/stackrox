//go:build compliance

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	complianceoperatorv1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func skipUnlessComplianceFreshnessEnabled(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	resp, err := v1.NewFeatureFlagServiceClient(conn).GetFeatureFlags(ctx, &v1.Empty{})
	require.NoError(t, err, "failed to retrieve feature flags")

	envVar := features.ComplianceSurfaceStaleData.EnvVar()
	for _, flag := range resp.GetFeatureFlags() {
		if flag.GetEnvVar() == envVar {
			if !flag.GetEnabled() {
				t.Skipf("%s is disabled on Central", envVar)
			}
			return
		}
	}
	t.Skipf("feature flag %s not reported by Central", envVar)
}

// scheduleAt returns a daily schedule at t's UTC hour:minute. Central evaluates schedules
// against UTC regardless of its clock's timezone, so no UTC assumption about Central itself
// is needed.
func scheduleAt(t time.Time) *v2.Schedule {
	u := t.UTC()
	return &v2.Schedule{
		Hour:         int32(u.Hour()),
		Minute:       int32(u.Minute()),
		IntervalType: v2.Schedule_DAILY,
	}
}

// scanConfigQuery scopes a results query to scanConfigName. GetComplianceProfileClusterResults is
// keyed by profile+cluster, not scan config, so without this scope a read sees rows from any other
// config that has used the same profile on the cluster (including deleted ones, cleaned up async).
func scanConfigQuery(scanConfigName string, extra ...string) *v2.RawQuery {
	q := search.ComplianceOperatorScanConfigName.String() + ":" + scanConfigName
	for _, e := range extra {
		q += "+" + e
	}
	return &v2.RawQuery{Query: q}
}

// waitForCheckResultsFreshness polls until every check result of profile+scanConfigName reaches
// want and Central reports a last scan time (results and the scan-end update arrive as separate
// sensor messages), then returns the last response so callers can read server-side data.
func waitForCheckResultsFreshness(t *testing.T, ctx context.Context, client v2.ComplianceResultsServiceClient,
	clusterID, profile, scanConfigName string, want v2.ComplianceCheckDataFreshness, timeout, interval time.Duration,
) *v2.ListComplianceCheckResultResponse {
	t.Helper()
	t.Logf("Waiting up to %s for check results of profile %s (scan config %s) on cluster %s to reach freshness %s",
		timeout, profile, scanConfigName, clusterID, want)
	var mu sync.Mutex
	var last *v2.ListComplianceCheckResultResponse
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := client.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: profile,
			ClusterId:   clusterID,
			Query:       scanConfigQuery(scanConfigName),
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", profile, clusterID)
		require.NotEmptyf(c, resp.GetCheckResults(), "no check results yet for profile %s cluster %s scan config %s", profile, clusterID, scanConfigName)
		for _, r := range resp.GetCheckResults() {
			require.Equalf(c, want, r.GetDataFreshness(), "check %s freshness: got %s, want %s",
				r.GetCheckName(), r.GetDataFreshness(), want)
		}
		require.NotNilf(c, resp.GetLastScanTime(),
			"Central has not reported a last scan time for profile %s cluster %s yet", profile, clusterID)
		concurrency.WithLock(&mu, func() { last = resp })
	}, timeout, interval)
	t.Logf("Check results of profile %s (scan config %s) on cluster %s reached freshness %s", profile, scanConfigName, clusterID, want)
	return concurrency.WithLock1(&mu, func() *v2.ListComplianceCheckResultResponse { return last })
}

// getComplianceProfileStats returns the single stats banner entry for profile. The banner is
// profile-global, not scoped to a scan config, so callers must use >= bounds, not exact counts.
// Exactly one entry is still expected since a profile maps to one scan config cluster-wide.
func getComplianceProfileStats(t require.TestingT, ctx context.Context, client v2.ComplianceResultsStatsServiceClient, profile string) *v2.ComplianceProfileScanStats {
	resp, err := client.GetComplianceProfileStats(ctx, &v2.ComplianceProfileResultsRequest{ProfileName: profile})
	require.NoErrorf(t, err, "failed to get compliance profile stats for %s", profile)
	require.Lenf(t, resp.GetScanStats(), 1, "expected exactly one scan stats entry for profile %s", profile)
	return resp.GetScanStats()[0]
}

// scaleSensorDownForTest scales the cluster's sensor to 0 and restores it to 1 on cleanup.
func scaleSensorDownForTest(ctx context.Context, t *testing.T, k8sClient kubernetes.Interface, dynClient ctrlClient.Client) {
	t.Cleanup(func() {
		scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 1)
		waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 1)
		// Sensor's readiness probe succeeds unconditionally, so also wait until Central sees the
		// sensor again; otherwise the next test's first message to sensor fails.
		connCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		waitUntilCentralSensorConnectionIs(t, connCtx, storage.ClusterHealthStatus_HEALTHY)
	})
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 0)
	// A scale-down only SIGTERMs the pod, which keeps relaying to Central for its full 30s
	// termination grace period while ReadyReplicas is already 0. Force-kill it so the cut is
	// immediate; an abrupt death is also the realistic failure mode these tests simulate.
	grace := int64(0)
	sensorPods, err := k8sClient.CoreV1().Pods(stackroxNamespace).List(ctx, metav1.ListOptions{LabelSelector: "app=sensor"})
	require.NoErrorf(t, err, "failed to list sensor pods in namespace %s", stackroxNamespace)
	for _, pod := range sensorPods.Items {
		require.NoErrorf(t, k8sClient.CoreV1().Pods(stackroxNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{GracePeriodSeconds: &grace}),
			"failed to force-delete sensor pod %s", pod.Name)
	}
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 0)
}

// Central's production freshness timings, from pkg/env/compliance_operator.go. Central uses them
// when the deployment leaves the env vars unset.
const (
	prodScanWatcherTimeout     = 40 * time.Minute
	prodScheduleWatcherTimeout = 45 * time.Minute
	prodSweepInterval          = time.Hour
)

// complianceFreshnessTimings is Central's deployed freshness timing configuration. The STALE
// tests derive their wait budgets from it because hard-coded budgets only hold for the shortened
// timings the compliance e2e job deploys Central with (SHORTEN_COMPLIANCE_FRESHNESS_TIMINGS in
// tests/e2e/lib.sh).
type complianceFreshnessTimings struct {
	// scanWatcherTimeout is ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT: how long Central's scan watcher
	// waits for a scan's results before it times out and marks the prior results STALE.
	scanWatcherTimeout time.Duration
	// grace is the sweeper's grace window: the scan and schedule watcher timeouts combined.
	// Nothing may flip results STALE before it has fully elapsed past a schedule tick.
	grace time.Duration
	// sweepInterval is how often the freshness sweeper runs, already clamped to Central's 1m
	// floor (minSweepInterval in central/complianceoperator/v2/freshness).
	sweepInterval time.Duration
}

// getFreshnessTimingsOrSkip reads the freshness timing env vars from the central deployment and
// skips the test when Central runs with the production defaults: the STALE waits would then take
// well over an hour, which only makes sense with shortened timings. Only the central deployment
// matters; central-db and central-worker replicas never run the sweeper or watchers.
func getFreshnessTimingsOrSkip(ctx context.Context, t *testing.T, k8sClient kubernetes.Interface) complianceFreshnessTimings {
	t.Helper()
	dep, err := k8sClient.AppsV1().Deployments(stackroxNamespace).Get(ctx, "central", metav1.GetOptions{})
	require.NoErrorf(t, err, "failed to get central deployment in namespace %s", stackroxNamespace)

	envVars := make(map[string]string)
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, e := range c.Env {
			envVars[e.Name] = e.Value
		}
	}
	// An unset env var means Central runs the production default for it.
	readDuration := func(name string, def time.Duration) time.Duration {
		v, ok := envVars[name]
		if !ok || v == "" {
			return def
		}
		d, err := time.ParseDuration(v)
		require.NoErrorf(t, err, "central deployment has invalid %s value %q", name, v)
		return d
	}
	scanWatcher := readDuration("ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT", prodScanWatcherTimeout)
	scheduleWatcher := readDuration("ROX_COMPLIANCE_SCAN_SCHEDULE_WATCHER_TIMEOUT", prodScheduleWatcherTimeout)
	sweepInterval := readDuration("ROX_COMPLIANCE_FRESHNESS_SWEEP_INTERVAL", prodSweepInterval)

	if scanWatcher == prodScanWatcherTimeout && scheduleWatcher == prodScheduleWatcherTimeout && sweepInterval == prodSweepInterval {
		t.Skipf("Central runs with production freshness timings (grace %s, sweep interval %s); "+
			"this test needs shortened ones. Deploy Central with ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT, "+
			"ROX_COMPLIANCE_SCAN_SCHEDULE_WATCHER_TIMEOUT and ROX_COMPLIANCE_FRESHNESS_SWEEP_INTERVAL set "+
			"to short values (see SHORTEN_COMPLIANCE_FRESHNESS_TIMINGS in tests/e2e/lib.sh)",
			prodScanWatcherTimeout+prodScheduleWatcherTimeout, prodSweepInterval)
	}
	return complianceFreshnessTimings{
		scanWatcherTimeout: scanWatcher,
		grace:              scanWatcher + scheduleWatcher,
		sweepInterval:      max(sweepInterval, time.Minute),
	}
}

// createFreshnessProfile creates a from-scratch TailoredProfile with a single CEL CustomRule and
// waits until Central knows it, returning the name to reference in a scan config. One rule keeps
// a scan down to seconds, which is what lets these tests run with short watcher timeouts; each
// test gets its own profile, so no two scan configs race for one profile name.
func createFreshnessProfile(ctx context.Context, t *testing.T, dynClient ctrlClient.Client,
	profileClient v2.ComplianceProfileServiceClient, clusterID, name string,
) string {
	createCustomRule(ctx, t, dynClient, name)
	tp := &complianceoperatorv1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: coNamespaceV2},
		Spec: complianceoperatorv1.TailoredProfileSpec{
			Title:       fmt.Sprintf("E2E freshness profile %s", name),
			Description: "From-scratch TailoredProfile with one CEL custom rule for freshness e2e",
			EnableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: name, Kind: complianceoperatorv1.CustomRuleKind, Rationale: "e2e test"},
			},
		},
	}
	require.NoErrorf(t, dynClient.Create(ctx, tp), "failed to create TailoredProfile %s", name)
	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, dynClient, name, coNamespaceV2)
	})
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.TailoredProfile
		require.NoErrorf(c, dynClient.Get(ctx, types.NamespacedName{Name: name, Namespace: coNamespaceV2}, &current),
			"failed to get TailoredProfile %s", name)
		require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
			"TailoredProfile %s not READY (state: %q, error: %q)", name, current.Status.State, current.Status.ErrorMessage)
	}, 10*time.Second, 1*time.Second)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, name)
	return name
}

// TestComplianceV2FreshnessCurrent proves a normal scan run's check results are reported CURRENT,
// never STALE. Must run non-parallel: other tests in this suite scale Sensor to 0, which would
// starve this test's scan of sensor updates.
func TestComplianceV2FreshnessCurrent(t *testing.T) {
	skipUnlessComplianceFreshnessEnabled(t)
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	resultsClient := v2.NewComplianceResultsServiceClient(conn)
	clusterID := getClusterID(t)

	testID := fmt.Sprintf("freshness-current-%s", uuid.NewV4().String())
	profileName := createFreshnessProfile(ctx, t, dynClient, v2.NewComplianceProfileServiceClient(conn), clusterID, testID)
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{profileName},
			Description: "freshness current e2e test",
			// A tick in the recent past won't recur for ~24h, so no scheduled run or sweeper tick
			// can interfere with this test.
			ScanSchedule: scheduleAt(time.Now().Add(-5 * time.Minute)),
		},
	}
	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, deleteScanConfig(ctx, resp.GetId(), scanConfigService), "failed to delete scan config %s", testID)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	waitForComplianceSuiteToComplete(t, dynClient, resp.GetScanName(), waitForDoneInterval, waitForDoneTimeout)

	currentResults := waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, profileName, testID,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)
	wantCurrentCount := len(currentResults.GetCheckResults())

	staleResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: profileName,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:STALE"),
	})
	require.NoError(t, err)
	assert.Empty(t, staleResp.GetCheckResults(), "expected no STALE rows for this scan config")

	// The STALE assertion above would also pass if the freshness filter matched nothing at all,
	// so prove the filter works by requiring the CURRENT side to return every row.
	currentResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: profileName,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:CURRENT"),
	})
	require.NoError(t, err)
	assert.Lenf(t, currentResp.GetCheckResults(), wantCurrentCount, "expected all %d checks of this scan config to be CURRENT", wantCurrentCount)
}

// TestComplianceV2FreshnessStaleViaSweeper proves the freshness sweeper flips CURRENT results to
// STALE once a schedule tick passes with no refresh. Must run non-parallel: it scales Sensor to 0.
func TestComplianceV2FreshnessStaleViaSweeper(t *testing.T) {
	skipUnlessComplianceFreshnessEnabled(t)
	ctx := context.Background()
	k8sClient := createK8sClient(t)
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	resultsClient := v2.NewComplianceResultsServiceClient(conn)
	statsClient := v2.NewComplianceResultsStatsServiceClient(conn)
	clusterID := getClusterID(t)
	timings := getFreshnessTimingsOrSkip(ctx, t, k8sClient)

	testID := fmt.Sprintf("freshness-sweeper-%s", uuid.NewV4().String())
	profileName := createFreshnessProfile(ctx, t, dynClient, v2.NewComplianceProfileServiceClient(conn), clusterID, testID)
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{profileName},
			Description: "freshness sweeper e2e test",
			// A tick in the recent past won't recur for ~24h, so nothing can flip results STALE
			// before the deliberate tick is set below.
			ScanSchedule: scheduleAt(time.Now().Add(-5 * time.Minute)),
		},
	}
	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, deleteScanConfig(ctx, resp.GetId(), scanConfigService), "failed to delete scan config %s", testID)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	waitForComplianceSuiteToComplete(t, dynClient, resp.GetScanName(), waitForDoneInterval, waitForDoneTimeout)
	currentResults := waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, profileName, testID,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)
	wantStaleCount := int32(len(currentResults.GetCheckResults()))

	// Nothing can refresh the results from this point on.
	scaleSensorDownForTest(ctx, t, k8sClient, dynClient)

	// Pick a near-future tick just before the update: the sweeper rejects ticks that predate the
	// config's LastUpdatedTime, so a past tick would not recur for ~24h, outside this test's window.
	tick := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
	if time.Until(tick) < 15*time.Second {
		// Ensure the update lands before the tick.
		tick = tick.Add(time.Minute)
	}
	updateReq := req.CloneVT()
	updateReq.Id = resp.GetId()
	updateReq.ScanConfig.ScanSchedule = scheduleAt(tick)
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, updateReq)
	require.NoErrorf(t, err, "failed to update scan config %s schedule", testID)

	// Verify against Central's own view that the update landed before the tick, since the sweeper
	// rejects ticks that predate LastUpdatedTime.
	updatedCfg, err := scanConfigService.GetComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: resp.GetId()})
	require.NoErrorf(t, err, "failed to read back scan config %s", testID)
	require.NotNilf(t, updatedCfg.GetLastUpdatedTime(),
		"scan config %s has no last updated time; cannot verify the update landed before tick %s", testID, tick)
	require.Falsef(t, updatedCfg.GetLastUpdatedTime().AsTime().After(tick),
		"scan config %s was updated at %s, after the chosen tick %s; the sweeper would reject the tick",
		testID, updatedCfg.GetLastUpdatedTime().AsTime(), tick)

	// Lower bound: the grace window must be respected. Nothing may mark results STALE before
	// tick + grace (see getFreshnessTimingsOrSkip) has fully elapsed. Sample after at least one
	// sweep interval has elapsed past the tick but before the grace window ends: a sweeper that
	// ignored grace would have marked by then, so everything must still be CURRENT here.
	time.Sleep(time.Until(tick.Add(max(timings.sweepInterval, timings.grace-30*time.Second))))
	preGraceResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: profileName,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID),
	})
	require.NoError(t, err)
	require.NotEmpty(t, preGraceResp.GetCheckResults())
	for _, r := range preGraceResp.GetCheckResults() {
		require.Equalf(t, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, r.GetDataFreshness(),
			"check %s turned STALE before the grace window elapsed", r.GetCheckName())
	}

	// The sweeper flips results to STALE once now > tick + grace (see getFreshnessTimingsOrSkip)
	// and detects it on its next sweep. The budget below covers the wait for the tick, the grace
	// window, one sweep interval and slack.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resultsResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: profileName,
			ClusterId:   clusterID,
			Query:       scanConfigQuery(testID),
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", profileName, clusterID)
		require.NotEmpty(c, resultsResp.GetCheckResults(), "no check results found")
		for _, r := range resultsResp.GetCheckResults() {
			require.Equalf(c, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE, r.GetDataFreshness(),
				"check %s freshness: got %s, want STALE", r.GetCheckName(), r.GetDataFreshness())
		}

		// Banner is profile-global (see getComplianceProfileStats), so assert lower bounds only.
		stats := getComplianceProfileStats(c, ctx, statsClient, profileName)
		require.GreaterOrEqual(c, stats.GetStaleClusterCount(), int32(1))
		require.GreaterOrEqual(c, stats.GetStaleCheckResultCount(), wantStaleCount)
	}, time.Until(tick)+timings.grace+timings.sweepInterval+time.Minute, 15*time.Second)

	staleResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: profileName,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:STALE"),
	})
	require.NoError(t, err)
	assert.Lenf(t, staleResp.GetCheckResults(), int(wantStaleCount), "expected all %d checks of this scan config to be STALE", wantStaleCount)

	currentResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: profileName,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:CURRENT"),
	})
	require.NoError(t, err)
	assert.Empty(t, currentResp.GetCheckResults(), "expected CURRENT filter (scoped to this scan config) to return no rows")
}

// TestComplianceV2FreshnessStaleViaScanWatcherTimeout proves a rescan that never delivers any
// check results marks its prior results STALE when its scan watcher times out. The test breaks
// the profile's CustomRule before rescanning: the cel-scanner exits on the structurally invalid
// rule without writing a CheckResult, so the run starts (Central sees the reset and creates the
// watcher) but can never deliver. A sensor cut cannot starve this run reliably: the one-rule
// scan delivers within seconds, faster than any observation-triggered cut. The schedule is set
// with no near-term tick so the sweeper can't also produce STALE here. Must run non-parallel:
// other tests in this suite scale Sensor to 0, which would starve this test's relays.
func TestComplianceV2FreshnessStaleViaScanWatcherTimeout(t *testing.T) {
	skipUnlessComplianceFreshnessEnabled(t)
	ctx := context.Background()
	k8sClient := createK8sClient(t)
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	resultsClient := v2.NewComplianceResultsServiceClient(conn)
	statsClient := v2.NewComplianceResultsStatsServiceClient(conn)
	clusterID := getClusterID(t)
	timings := getFreshnessTimingsOrSkip(ctx, t, k8sClient)

	scanStart := time.Now()
	testID := fmt.Sprintf("freshness-timeout-%s", uuid.NewV4().String())
	profileName := createFreshnessProfile(ctx, t, dynClient, v2.NewComplianceProfileServiceClient(conn), clusterID, testID)
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{profileName},
			Description: "freshness scan-watcher-timeout e2e test",
			// A tick in the recent past won't recur for ~24h, so it can't fire during this test.
			ScanSchedule: scheduleAt(scanStart.Add(-5 * time.Minute)),
		},
	}
	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, deleteScanConfig(ctx, resp.GetId(), scanConfigService), "failed to delete scan config %s", testID)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
		// The starved rescan's scan pod keeps restarting (RestartPolicy OnFailure) until the
		// suite is torn down, so wait for the suite and its scan to go -- the scan's finalizer
		// deletes the pod -- rather than leaving the crash loop running into later tests.
		deleteResource[complianceoperatorv1.ComplianceSuite](ctx, t, dynClient, testID, coNamespaceV2)
		deleteResource[complianceoperatorv1.ComplianceScan](ctx, t, dynClient, testID, coNamespaceV2)
	})

	waitForComplianceSuiteToComplete(t, dynClient, resp.GetScanName(), waitForDoneInterval, waitForDoneTimeout)
	currentResults := waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, profileName, testID,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)
	wantStaleCount := int32(len(currentResults.GetCheckResults()))
	priorCheckIDs := make([]string, 0, len(currentResults.GetCheckResults()))
	for _, r := range currentResults.GetCheckResults() {
		priorCheckIDs = append(priorCheckIDs, r.GetCheckId())
	}
	// A completed scan leaves a last scan time here (the helper above waited for it); the rescan
	// below clears it, which the later poll relies on to confirm Central ingested the reset.

	// Break the rule's inputs: the rescan's cel-scanner validates the rule payload's
	// structure before scanning -- an inputless rule fails that check and the scanner exits
	// without writing any CheckResult, so the run starts but can never deliver. (Breaking
	// the CEL expression instead would not do: the scanner only validates structure, and a
	// syntactically invalid expression yields an ERROR CheckResult that satisfies the
	// watcher.) The rule object must stay around -- deleting it would break Central's
	// results queries, which resolve every check result's rule. The sensor stays up and
	// relays the run's start, which is what creates the watcher whose timeout must mark
	// the prior results STALE.
	rule := &complianceoperatorv1.CustomRule{}
	require.NoErrorf(t, dynClient.Get(ctx, types.NamespacedName{Name: testID, Namespace: coNamespaceV2}, rule),
		"failed to get CustomRule %s", testID)
	rule.Spec.CustomRulePayload.Inputs = nil
	require.NoErrorf(t, dynClient.Update(ctx, rule), "failed to break CustomRule %s", testID)

	// Start a rescan that can never deliver results. Its watcher's prior results are the ones
	// that should end up STALE.
	_, err = scanConfigService.RunComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: resp.GetId()})
	require.NoErrorf(t, err, "failed to trigger rescan for %s", testID)

	// Poll until the profile's last scan time is nil: the rescan clears it, so nil proves
	// Central ingested the scan reset. (This does not by itself prove the watcher was created
	// -- the STALE assertion below is that proof.)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resultsResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: profileName,
			ClusterId:   clusterID,
			Query:       scanConfigQuery(testID),
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", profileName, clusterID)
		require.Nilf(c, resultsResp.GetLastScanTime(),
			"Central still reports last scan time %v; it has not observed the rescan's scan reset yet", resultsResp.GetLastScanTime())
	}, 2*time.Minute, 5*time.Second)

	// Central's scan watcher times out after ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT (see
	// getFreshnessTimingsOrSkip) and marks the prior results STALE. The budget below covers the
	// timeout plus slack for the watcher's cleanup.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resultsResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: profileName,
			ClusterId:   clusterID,
			Query:       scanConfigQuery(testID),
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", profileName, clusterID)
		require.NotEmpty(c, resultsResp.GetCheckResults(), "no check results found")
		for _, r := range resultsResp.GetCheckResults() {
			require.Equalf(c, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE, r.GetDataFreshness(),
				"check %s freshness: got %s, want STALE", r.GetCheckName(), r.GetDataFreshness())
		}

		// Banner is profile-global (see getComplianceProfileStats), so assert lower bounds only.
		stats := getComplianceProfileStats(c, ctx, statsClient, profileName)
		require.GreaterOrEqual(c, stats.GetStaleClusterCount(), int32(1))
		require.GreaterOrEqual(c, stats.GetStaleCheckResultCount(), wantStaleCount)
	}, timings.scanWatcherTimeout+2*time.Minute, 15*time.Second)

	// The STALE set must match the original set exactly, not merely be nonempty like the stats above.
	staleResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: profileName,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:STALE"),
	})
	require.NoError(t, err)
	staleCheckIDs := make([]string, 0, len(staleResp.GetCheckResults()))
	for _, r := range staleResp.GetCheckResults() {
		staleCheckIDs = append(staleCheckIDs, r.GetCheckId())
	}
	assert.ElementsMatch(t, priorCheckIDs, staleCheckIDs,
		"expected every one of the %d prior checks to survive and be marked STALE", len(priorCheckIDs))
}
