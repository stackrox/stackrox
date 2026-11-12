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
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// freshnessProfile is the profile used by every freshness test. A scan config's profile must be
// unique cluster-wide (see TestComplianceV2CreateGetScanConfigurations), and these tests are
// non-parallel, so they never race each other or TestComplianceV2CentralSendsScanConfiguration
// (also non-parallel) for it.
const freshnessProfile = "ocp4-cis"

// skipUnlessComplianceFreshnessEnabled skips the test unless Central reports the
// ComplianceSurfaceStaleData feature flag as enabled.
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

// scheduleAt returns a daily schedule firing at t's hour:minute (UTC). Central's freshness
// sweeper evaluates schedules against time.Now(), whose Location() gronx uses as-is (see
// previousScheduledStart in central/complianceoperator/v2/freshness/sweeper.go); this assumes
// Central's process clock is UTC, the default for StackRox's container images.
func scheduleAt(t time.Time) *v2.Schedule {
	u := t.UTC()
	return &v2.Schedule{
		Hour:         int32(u.Hour()),
		Minute:       int32(u.Minute()),
		IntervalType: v2.Schedule_DAILY,
	}
}

// waitForCheckResultsFreshness polls the per-check results of profile on clusterID until every
// result's DataFreshness equals want, or fails the test after timeout. It returns the last
// successful response so callers can read authoritative server-side data (e.g. LastScanTime).
func waitForCheckResultsFreshness(t *testing.T, ctx context.Context, client v2.ComplianceResultsServiceClient,
	clusterID, profile string, want v2.ComplianceCheckDataFreshness, timeout, interval time.Duration,
) *v2.ListComplianceCheckResultResponse {
	t.Helper()
	t.Logf("Waiting up to %s for check results of profile %s on cluster %s to reach freshness %s",
		timeout, profile, clusterID, want)
	var mu sync.Mutex
	var last *v2.ListComplianceCheckResultResponse
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := client.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: profile,
			ClusterId:   clusterID,
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", profile, clusterID)
		require.NotEmptyf(c, resp.GetCheckResults(), "no check results yet for profile %s cluster %s", profile, clusterID)
		for _, r := range resp.GetCheckResults() {
			require.Equalf(c, want, r.GetDataFreshness(), "check %s freshness: got %s, want %s",
				r.GetCheckName(), r.GetDataFreshness(), want)
		}
		concurrency.WithLock(&mu, func() { last = resp })
	}, timeout, interval)
	t.Logf("Check results of profile %s on cluster %s reached freshness %s", profile, clusterID, want)
	return last
}

// getComplianceProfileStats returns the single stats banner entry for profile. A scan config's
// profile is unique cluster-wide, so exactly one entry is expected.
func getComplianceProfileStats(t require.TestingT, ctx context.Context, client v2.ComplianceResultsStatsServiceClient, profile string) *v2.ComplianceProfileScanStats {
	resp, err := client.GetComplianceProfileStats(ctx, &v2.ComplianceProfileResultsRequest{ProfileName: profile})
	require.NoErrorf(t, err, "failed to get compliance profile stats for %s", profile)
	require.Lenf(t, resp.GetScanStats(), 1, "expected exactly one scan stats entry for profile %s", profile)
	return resp.GetScanStats()[0]
}

// waitForComplianceSuiteRunning waits for the ComplianceSuite CR to reach the RUNNING phase, used
// to detect that a rescan has started on the cluster.
func waitForComplianceSuiteRunning(t *testing.T, client ctrlClient.Client, suiteName string, interval, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	t.Logf("Waiting for ComplianceSuite %s to reach RUNNING phase", suiteName)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		callCtx, callCancel := context.WithTimeout(ctx, interval)
		defer callCancel()

		var suite complianceoperatorv1.ComplianceSuite
		err := client.Get(callCtx, types.NamespacedName{Name: suiteName, Namespace: coNamespaceV2}, &suite)
		require.NoErrorf(c, err, "failed to get ComplianceSuite %s", suiteName)
		require.Equalf(c, complianceoperatorv1.PhaseRunning, suite.Status.Phase,
			"ComplianceSuite %s not RUNNING (current phase is %q)", suiteName, suite.Status.Phase)
	}, timeout, interval)
	t.Logf("ComplianceSuite %s has reached RUNNING phase", suiteName)
}

// scaleSensorDownForTest scales the single cluster's sensor to 0 and registers a cleanup that
// scales it back to 1 and waits for it to be ready, so subsequent tests get a healthy sensor.
func scaleSensorDownForTest(ctx context.Context, t *testing.T, k8sClient kubernetes.Interface, dynClient ctrlClient.Client) {
	t.Cleanup(func() {
		scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 1)
		waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 1)
	})
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 0)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 0)
}

// TestComplianceV2FreshnessCurrent verifies that check results delivered by a normal scan run are
// reported CURRENT and that the stale-data banner stats are zero.
//
// Run outside of other parallel tests: freshnessProfile can only be used by one scan config
// cluster-wide at a time.
func TestComplianceV2FreshnessCurrent(t *testing.T) {
	skipUnlessComplianceFreshnessEnabled(t)
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	resultsClient := v2.NewComplianceResultsServiceClient(conn)
	statsClient := v2.NewComplianceResultsStatsServiceClient(conn)
	clusterID := getClusterID(t)

	testID := fmt.Sprintf("freshness-current-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{freshnessProfile},
			Description:  "freshness current e2e test",
			ScanSchedule: initialSchedule,
		},
	}
	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	_, err = scanConfigService.RunComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: resp.GetId()})
	require.NoErrorf(t, err, "failed to run scan config %s", testID)
	waitForComplianceSuiteToComplete(t, dynClient, resp.GetScanName(), waitForDoneInterval, waitForDoneTimeout)

	waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, freshnessProfile,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)

	stats := getComplianceProfileStats(t, ctx, statsClient, freshnessProfile)
	assert.EqualValues(t, 0, stats.GetStaleClusterCount())
	assert.EqualValues(t, 0, stats.GetStaleCheckResultCount())
}

// TestComplianceV2FreshnessStaleViaSweeper deterministically drives the freshness sweeper
// (central/complianceoperator/v2/freshness) to flip CURRENT results to STALE by updating the scan
// config's schedule to a tick shortly after the results' LastStartedTime, then cutting off the
// sensor so nothing can refresh them before the sweeper's grace window elapses.
//
// Run outside of other parallel tests: it scales the single cluster's Sensor to 0.
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

	testID := fmt.Sprintf("freshness-sweeper-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{freshnessProfile},
			Description:  "freshness sweeper e2e test",
			ScanSchedule: initialSchedule,
		},
	}
	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	_, err = scanConfigService.RunComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: resp.GetId()})
	require.NoErrorf(t, err, "failed to run scan config %s", testID)
	waitForComplianceSuiteToComplete(t, dynClient, resp.GetScanName(), waitForDoneInterval, waitForDoneTimeout)
	currentResults := waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, freshnessProfile,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)

	// Nothing can refresh the results from this point on.
	scaleSensorDownForTest(ctx, t, k8sClient, dynClient)

	// A daily tick 2 minutes after the scan's completion (always >= every result's
	// LastStartedTime, since a check result can't start after its scan finishes) gives the sweeper
	// an "expected refresh" scheduled after the results' LastStartedTime without waiting a full day
	// for the next occurrence; see previousScheduledStart in
	// central/complianceoperator/v2/freshness/sweeper.go. Deriving the tick from Central's own
	// LastScanTime (rather than a client-side guess of the scan's start time) avoids a tie: both the
	// schedule (minute granularity) and LastStartedTime (second granularity, stamped by the
	// Compliance Operator) truncate sub-minute precision, and CO can start a scan within a second of
	// the request, so a 1-minute, start-time-based guess can land in the same minute as the actual
	// LastStartedTime and never compare strictly less-than.
	lastScanTime := currentResults.GetLastScanTime().AsTime()
	updateReq := req.CloneVT()
	updateReq.Id = resp.GetId()
	updateReq.ScanConfig.ScanSchedule = scheduleAt(lastScanTime.Add(2 * time.Minute))
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, updateReq)
	require.NoErrorf(t, err, "failed to update scan config %s schedule", testID)

	// The sweeper flips results to STALE once now > (lastScanTime+2m) + grace, where grace is
	// ROX_COMPLIANCE_SCAN_SCHEDULE_WATCHER_TIMEOUT + ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT (shortened
	// for e2e to 1m+2m=3m, see tests/e2e/lib.sh and deploy/common/k8sbased.sh).
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resultsResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: freshnessProfile,
			ClusterId:   clusterID,
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", freshnessProfile, clusterID)
		require.NotEmpty(c, resultsResp.GetCheckResults(), "no check results found")
		for _, r := range resultsResp.GetCheckResults() {
			require.Equalf(c, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE, r.GetDataFreshness(),
				"check %s freshness: got %s, want STALE", r.GetCheckName(), r.GetDataFreshness())
		}

		stats := getComplianceProfileStats(c, ctx, statsClient, freshnessProfile)
		require.EqualValues(c, 1, stats.GetStaleClusterCount())
		require.Greater(c, stats.GetStaleCheckResultCount(), int32(0))
	}, 8*time.Minute, 15*time.Second)

	// Filtering on the freshness label reflects the same transition.
	staleResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: freshnessProfile,
		ClusterId:   clusterID,
		Query:       &v2.RawQuery{Query: "Compliance Check Data Freshness:STALE"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, staleResp.GetCheckResults(), "expected STALE filter to return rows")

	currentResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: freshnessProfile,
		ClusterId:   clusterID,
		Query:       &v2.RawQuery{Query: "Compliance Check Data Freshness:CURRENT"},
	})
	require.NoError(t, err)
	assert.Empty(t, currentResp.GetCheckResults(), "expected CURRENT filter to return no rows")
}

// TestComplianceV2FreshnessStaleViaScanWatcherTimeout drives the write-time scan-watcher-timeout
// path (central/complianceoperator/v2/checkresults/datastore MarkScanChecksStale, invoked from
// central/complianceoperator/v2/report/manager markPriorResultsStale) to mark a run's prior
// results STALE: a rescan is started and then starved of sensor updates until its scan watcher
// times out.
//
// The config's schedule is set to a time with no near-term tick so the sweeper
// (TestComplianceV2FreshnessStaleViaSweeper) cannot also produce STALE within this test's window.
//
// Run outside of other parallel tests: it scales the single cluster's Sensor to 0.
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

	scanStart := time.Now()
	testID := fmt.Sprintf("freshness-timeout-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{freshnessProfile},
			Description: "freshness scan-watcher-timeout e2e test",
			// A tick in the recent past won't recur for ~24h, so it can't fire during this test.
			ScanSchedule: scheduleAt(scanStart.Add(-5 * time.Minute)),
		},
	}
	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	_, err = scanConfigService.RunComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: resp.GetId()})
	require.NoErrorf(t, err, "failed to run scan config %s", testID)
	waitForComplianceSuiteToComplete(t, dynClient, resp.GetScanName(), waitForDoneInterval, waitForDoneTimeout)
	waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, freshnessProfile,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)

	// Start a second scan that will never complete once the sensor is cut below; its watcher's
	// prior results (the ones above) are what should end up STALE.
	_, err = scanConfigService.RunComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: resp.GetId()})
	require.NoErrorf(t, err, "failed to trigger rescan for %s", testID)

	// shortcut: timing-sensitive. Central only creates the scan watcher once sensor relays a Scan
	// or CheckResult update for the rescan, which can lag the CR's RUNNING phase. We wait for
	// RUNNING as a proxy, then sleep a fixed buffer for the relay. If this test becomes flaky,
	// increase the buffer rather than removing it.
	waitForComplianceSuiteRunning(t, dynClient, resp.GetScanName(), 5*time.Second, 2*time.Minute)
	time.Sleep(15 * time.Second)

	scaleSensorDownForTest(ctx, t, k8sClient, dynClient)

	// Central's scan watcher times out after ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT (shortened to 2m
	// for e2e, see tests/e2e/lib.sh and deploy/common/k8sbased.sh) and marks the prior results
	// (from the first, completed scan) STALE.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resultsResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: freshnessProfile,
			ClusterId:   clusterID,
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", freshnessProfile, clusterID)
		require.NotEmpty(c, resultsResp.GetCheckResults(), "no check results found")
		for _, r := range resultsResp.GetCheckResults() {
			require.Equalf(c, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE, r.GetDataFreshness(),
				"check %s freshness: got %s, want STALE", r.GetCheckName(), r.GetDataFreshness())
		}

		stats := getComplianceProfileStats(c, ctx, statsClient, freshnessProfile)
		require.EqualValues(c, 1, stats.GetStaleClusterCount())
	}, 4*time.Minute, 15*time.Second)
}
