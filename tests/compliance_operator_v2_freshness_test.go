//go:build compliance

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// scanConfigQuery scopes a per-check results query to scanConfigName, optionally ANDed with extra
// clauses (e.g. a freshness filter). GetComplianceProfileClusterResults is keyed by profile+cluster,
// not by scan config, so without this scope a read can see rows left behind by any other scan
// config that has ever used the same profile on the same cluster (including already-deleted ones,
// since check-result deletion on scan-config delete is asynchronous).
func scanConfigQuery(scanConfigName string, extra ...string) *v2.RawQuery {
	q := search.ComplianceOperatorScanConfigName.String() + ":" + scanConfigName
	for _, e := range extra {
		q += "+" + e
	}
	return &v2.RawQuery{Query: q}
}

// waitForCheckResultsFreshness polls the per-check results of profile+scanConfigName on clusterID
// until every result's DataFreshness equals want, or fails the test after timeout. It returns the
// last successful response so callers can read authoritative server-side data (e.g. LastScanTime).
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
		concurrency.WithLock(&mu, func() { last = resp })
	}, timeout, interval)
	t.Logf("Check results of profile %s (scan config %s) on cluster %s reached freshness %s", profile, scanConfigName, clusterID, want)
	return last
}

// getComplianceProfileStats returns the single stats banner entry for profile. The banner is
// profile-global (not scoped to a scan config), so callers must not assert exact counts under
// potential cross-test contamination; use >= bounds derived from this test's own known result
// count instead. A scan config's profile is unique cluster-wide at any point in time, so exactly
// one scanStats entry is still expected.
func getComplianceProfileStats(t require.TestingT, ctx context.Context, client v2.ComplianceResultsStatsServiceClient, profile string) *v2.ComplianceProfileScanStats {
	resp, err := client.GetComplianceProfileStats(ctx, &v2.ComplianceProfileResultsRequest{ProfileName: profile})
	require.NoErrorf(t, err, "failed to get compliance profile stats for %s", profile)
	require.Lenf(t, resp.GetScanStats(), 1, "expected exactly one scan stats entry for profile %s", profile)
	return resp.GetScanStats()[0]
}

// waitForScanConfigRescanObserved waits until Central's OWN view of scanConfigID (via
// GetComplianceScanConfiguration, not the raw Compliance Operator CR) shows clusterID's suite
// phase has moved past previousPhase. This is a stronger "the rescan started" signal than polling
// the Compliance Operator CRD directly: Central only reports a new phase here once sensor has
// relayed a ComplianceSuite update for it, and that relay is what also feeds the scan/check-result
// pipeline that creates the scan watcher (central/complianceoperator/v2/report/manager/watcher).
func waitForScanConfigRescanObserved(t *testing.T, ctx context.Context, client v2.ComplianceScanConfigurationServiceClient,
	scanConfigID, clusterID, previousPhase string, timeout, interval time.Duration,
) {
	t.Helper()
	t.Logf("Waiting up to %s for Central to observe scan config %s leave suite phase %q on cluster %s",
		timeout, scanConfigID, previousPhase, clusterID)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		status, err := client.GetComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: scanConfigID})
		require.NoErrorf(c, err, "failed to get scan config %s", scanConfigID)
		for _, cs := range status.GetClusterStatus() {
			if cs.GetClusterId() != clusterID {
				continue
			}
			phase := cs.GetSuiteStatus().GetPhase()
			require.NotEmptyf(c, phase, "Central has not reported a suite phase for cluster %s yet", clusterID)
			require.NotEqualf(c, previousPhase, phase, "Central still reports suite phase %q for cluster %s", previousPhase, clusterID)
			return
		}
		require.Failf(c, "cluster status not found", "cluster %s not found in scan config %s status", clusterID, scanConfigID)
	}, timeout, interval)
	t.Logf("Central observed scan config %s leave suite phase %q on cluster %s", scanConfigID, previousPhase, clusterID)
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
// reported CURRENT and that no row of this test's own scan config is ever reported STALE.
//
// It does not assert on the profile-wide stale-data banner (GetComplianceProfileStats): that
// banner is not scoped to a scan config, so it can't be made robust against contamination from
// other tests' (or pre-existing) ocp4-cis rows on this cluster. CURRENT-ness is instead proven via
// per-check results and a filter, both scoped to this test's scan config name.
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

	waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, freshnessProfile, testID,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)

	staleResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: freshnessProfile,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:STALE"),
	})
	require.NoError(t, err)
	assert.Empty(t, staleResp.GetCheckResults(), "expected no STALE rows for this scan config")
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
	currentResults := waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, freshnessProfile, testID,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)
	wantStaleCount := int32(len(currentResults.GetCheckResults()))

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
	// Scoped to this scan config, so the per-check assertion below can't see rows left behind by
	// other configs that have used (or are using) the same profile on this cluster.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resultsResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: freshnessProfile,
			ClusterId:   clusterID,
			Query:       scanConfigQuery(testID),
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", freshnessProfile, clusterID)
		require.NotEmpty(c, resultsResp.GetCheckResults(), "no check results found")
		for _, r := range resultsResp.GetCheckResults() {
			require.Equalf(c, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE, r.GetDataFreshness(),
				"check %s freshness: got %s, want STALE", r.GetCheckName(), r.GetDataFreshness())
		}

		// The banner (GetComplianceProfileStats) is profile-global, not scoped to a scan config, so
		// it can include stale rows from other configs that have used this profile. Assert lower
		// bounds derived from this test's own known result count rather than exact/zero values.
		stats := getComplianceProfileStats(c, ctx, statsClient, freshnessProfile)
		require.GreaterOrEqual(c, stats.GetStaleClusterCount(), int32(1))
		require.GreaterOrEqual(c, stats.GetStaleCheckResultCount(), wantStaleCount)
	}, 8*time.Minute, 15*time.Second)

	// Filtering on the freshness label, scoped to this scan config, reflects the same transition
	// precisely: every one of this config's checks is STALE and none is CURRENT.
	staleResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: freshnessProfile,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:STALE"),
	})
	require.NoError(t, err)
	assert.Lenf(t, staleResp.GetCheckResults(), int(wantStaleCount), "expected all %d checks of this scan config to be STALE", wantStaleCount)

	currentResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
		ProfileName: freshnessProfile,
		ClusterId:   clusterID,
		Query:       scanConfigQuery(testID, "Compliance Check Data Freshness:CURRENT"),
	})
	require.NoError(t, err)
	assert.Empty(t, currentResp.GetCheckResults(), "expected CURRENT filter (scoped to this scan config) to return no rows")
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
	currentResults := waitForCheckResultsFreshness(t, ctx, resultsClient, clusterID, freshnessProfile, testID,
		v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT, waitForDoneTimeout, waitForDoneInterval)
	wantStaleCount := int32(len(currentResults.GetCheckResults()))

	// Start a second scan that will never complete once the sensor is cut below; its watcher's
	// prior results (the ones above) are what should end up STALE.
	_, err = scanConfigService.RunComplianceScanConfiguration(ctx, &v2.ResourceByID{Id: resp.GetId()})
	require.NoErrorf(t, err, "failed to trigger rescan for %s", testID)

	// Wait for Central's OWN view of the suite to move off "DONE" (its phase after the first scan)
	// before cutting the sensor: Central only reports that once sensor has relayed a new
	// ComplianceSuite update, which is driven by the same sensor-relay mechanism that feeds the
	// scan/check-result pipeline creating the scan watcher. This is a stronger signal than polling
	// the Compliance Operator CRD directly, which only confirms the k8s-side state and says nothing
	// about whether Central has ingested it yet.
	waitForScanConfigRescanObserved(t, ctx, scanConfigService, resp.GetId(), clusterID, "DONE", 2*time.Minute, 5*time.Second)
	// Small fixed buffer: the suite and scan/check-result updates are relayed as separate messages
	// from the same sensor reconcile cycle, so observing the suite move off DONE does not strictly
	// guarantee the scan message (which creates the watcher) has also landed yet.
	time.Sleep(10 * time.Second)

	scaleSensorDownForTest(ctx, t, k8sClient, dynClient)

	// Central's scan watcher times out after ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT (shortened to 2m
	// for e2e, see tests/e2e/lib.sh and deploy/common/k8sbased.sh) and marks the prior results
	// (from the first, completed scan) STALE. Scoped to this scan config so the per-check assertion
	// can't see rows left behind by other configs that have used the same profile on this cluster.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resultsResp, err := resultsClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: freshnessProfile,
			ClusterId:   clusterID,
			Query:       scanConfigQuery(testID),
		})
		require.NoErrorf(c, err, "failed to get compliance results for profile %s cluster %s", freshnessProfile, clusterID)
		require.NotEmpty(c, resultsResp.GetCheckResults(), "no check results found")
		for _, r := range resultsResp.GetCheckResults() {
			require.Equalf(c, v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE, r.GetDataFreshness(),
				"check %s freshness: got %s, want STALE", r.GetCheckName(), r.GetDataFreshness())
		}

		// The banner is profile-global, not scoped to a scan config (see
		// TestComplianceV2FreshnessStaleViaSweeper); assert a lower bound, not an exact value.
		stats := getComplianceProfileStats(c, ctx, statsClient, freshnessProfile)
		require.GreaterOrEqual(c, stats.GetStaleClusterCount(), int32(1))
		require.GreaterOrEqual(c, stats.GetStaleCheckResultCount(), wantStaleCount)
	}, 4*time.Minute, 15*time.Second)
}
