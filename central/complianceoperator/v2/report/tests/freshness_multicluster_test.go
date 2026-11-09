//go:build sql_integration

package tests

import (
	"context"
	"testing"
	"time"

	framework "github.com/stackrox/rox/central/tests"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFreshnessHooksIsolatePerCluster proves that the write-time compliance-freshness hooks
// (ROX-35712, PR #23765) track CURRENT/STALE state independently per cluster: when one cluster's
// scan run times out, only that cluster's un-refreshed check results go STALE. A sibling cluster
// sharing the same scan configuration and profile, whose scan completes normally in the same
// cycle, must stay CURRENT throughout.
//
// This guards against the hooks accidentally scoping stale-marking by scan config or scan name
// (shared across clusters) instead of by scan ref (which is derived per cluster+scan name, see
// internaltov2storage.BuildNameRefID), which would incorrectly stale a healthy cluster's data.
// Cycle 2 deliberately staggers each cluster's scan timestamp (see the comment at cycle 2 below)
// so a scan-name-scoped bug cannot hide behind the stale-marking timestamp filter alone.
//
// Assertions read storage.ComplianceOperatorCheckResultV2.DataFreshness directly via
// CheckResultsDS, not through the v2 API: as of this writing, the v2 read path
// (storagetov2.checkResult) never copies DataFreshness onto the returned
// v2.ComplianceCheckResult, so GetComplianceScanResults always reports UNKNOWN regardless of the
// stored value. That is a separate, pre-existing gap in the read side and is reported alongside
// this PoC rather than patched here.
func TestFreshnessHooksIsolatePerCluster(t *testing.T) {
	t.Setenv(features.ComplianceSurfaceStaleData.EnvVar(), "true")
	t.Setenv(env.ComplianceScanWatcherTimeout.EnvVar(), "10s")
	t.Setenv(env.ComplianceScanScheduleWatcherTimeout.EnvVar(), "15s")

	c := framework.NewCentral(t)
	c.Start()

	healthySensor := framework.NewFakeSensor(t, "healthy-cluster")
	healthySensor.ConnectTo(c)
	timeoutSensor := framework.NewFakeSensor(t, "timeout-cluster")
	timeoutSensor.ConnectTo(c)

	conn := c.APIConn()
	const (
		profileID      = "xccdf_org.ssgproject.content_profile_cis"
		profileName    = "ocp4-cis"
		scanName       = profileName
		scanConfigName = "freshness-isolation-test"
		checkID        = "xccdf_org.ssgproject.content_rule_api_server_audit_log_maxbackup"
		checkName      = "ocp4-api-server-audit-log-maxbackup"
	)

	for _, s := range []*framework.FakeSensor{healthySensor, timeoutSensor} {
		framework.InjectComplianceOperatorInfo(s, "v1.6.0")
		framework.InjectProfile(s, uuid.NewV4().String(), profileName, profileID, "0.1.0", "CIS OpenShift Benchmark", "Platform")
		framework.InjectRule(s, uuid.NewV4().String(), checkName, "Rule",
			map[string]string{"compliance.openshift.io/profile": profileName})
	}

	profileClient := v2.NewComplianceProfileServiceClient(conn)
	for _, s := range []*framework.FakeSensor{healthySensor, timeoutSensor} {
		assert.Eventually(t, func() bool {
			resp, err := profileClient.ListComplianceProfiles(context.Background(),
				&v2.ProfilesForClusterRequest{ClusterId: s.ClusterID()})
			return err == nil && len(resp.GetProfiles()) >= 1
		}, 5*time.Second, 100*time.Millisecond, "profile should be ingested for cluster %s", s.ClusterID())
	}

	scanConfigClient := v2.NewComplianceScanConfigurationServiceClient(conn)
	createdScanConfig, err := scanConfigClient.CreateComplianceScanConfiguration(context.Background(),
		&v2.ComplianceScanConfiguration{
			ScanName: scanConfigName,
			Clusters: []string{healthySensor.ClusterID(), timeoutSensor.ClusterID()},
			ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
				OneTimeScan: true,
				Profiles:    []string{profileName},
			},
		})
	require.NoError(t, err)
	require.NotEmpty(t, createdScanConfig.GetId())

	// Query storage directly (see doc comment above for why: the v2 API doesn't surface
	// DataFreshness yet). One stable result ID per cluster (below) means UpsertResult always
	// updates the same row, so at most one row per cluster is ever stored.
	allAccessCtx := sac.WithAllAccess(context.Background())
	freshnessByCluster := func() map[string]storage.ComplianceOperatorCheckResultV2_DataFreshness {
		results, err := c.CheckResultsDS.SearchComplianceCheckResults(allAccessCtx, search.EmptyQuery())
		require.NoError(t, err)
		out := map[string]storage.ComplianceOperatorCheckResultV2_DataFreshness{}
		for _, r := range results {
			out[r.GetClusterId()] = r.GetDataFreshness()
		}
		return out
	}

	// waitForScanStartTime polls the scan datastore directly rather than resending the
	// check-result event in a loop. Resending would call HandleResult repeatedly, and each call
	// can spin up a brand-new in-memory ScanWatcher (manager_impl.go's HandleResult always calls
	// getWatcher(..., 0), which creates one whenever none is currently tracked for that ID). A
	// stray watcher from an earlier retry can outlive the real one and time out on its own clock,
	// marking results stale at the wrong moment and making STALE assertions flaky. Waiting for
	// the scan to land first means each check result only needs to be sent once.
	waitForScanStartTime := func(scanID string, want time.Time) {
		assert.Eventually(t, func() bool {
			scan, found, err := c.ScansDS.GetScan(allAccessCtx, scanID)
			require.NoError(t, err)
			return found && scan.GetLastStartedTime().AsTime().Equal(want)
		}, 8*time.Second, 100*time.Millisecond, "scan %s should be stored with start time %s", scanID, want)
	}

	// Stable per-cluster result ID: the storage row ID is the sensor-event ID
	// (internaltov2storage.ComplianceOperatorCheckResult), so re-injecting with the same ID
	// upserts the existing row instead of creating a new one -- matching how the real
	// Compliance Operator re-reports the same Kubernetes check-result object across scan runs.
	healthyResultID := uuid.NewV4().String()
	timeoutResultID := uuid.NewV4().String()

	// =====================================================================
	// Cycle 1 (T1): both clusters' scans complete normally. Both results
	// become CURRENT.
	// =====================================================================
	t1 := time.Date(2025, 5, 7, 10, 0, 0, 0, time.UTC)
	healthyScan1ID := uuid.NewV4().String()
	timeoutScan1ID := uuid.NewV4().String()
	framework.InjectScan(healthySensor, healthyScan1ID, scanName, profileID, scanConfigName, t1, -1)
	framework.InjectScan(timeoutSensor, timeoutScan1ID, scanName, profileID, scanConfigName, t1, -1)
	// The check-result pipeline requires the scan to already be stored (it looks up the scan by
	// ScanRefId), so wait for both scans to land before sending either check result -- see
	// waitForScanStartTime's doc comment for why each result is sent exactly once rather than
	// retried.
	waitForScanStartTime(healthyScan1ID, t1)
	waitForScanStartTime(timeoutScan1ID, t1)
	framework.InjectCheckResult(healthySensor, healthyResultID, checkID, checkName,
		scanName, scanConfigName, checkName, central.ComplianceOperatorCheckResultV2_PASS, t1)
	framework.InjectCheckResult(timeoutSensor, timeoutResultID, checkID, checkName,
		scanName, scanConfigName, checkName, central.ComplianceOperatorCheckResultV2_PASS, t1)
	assert.Eventually(t, func() bool {
		freshness := freshnessByCluster()
		_, healthyLanded := freshness[healthySensor.ClusterID()]
		_, timeoutLanded := freshness[timeoutSensor.ClusterID()]
		return healthyLanded && timeoutLanded
	}, 8*time.Second, 200*time.Millisecond, "cycle 1 check results should land")
	// Update the same scan objects with the check-count annotation to signal the watchers they
	// are done (reusing the scan IDs above, as the real Compliance Operator updates the same Scan
	// object in place rather than creating a new one).
	framework.InjectScan(healthySensor, healthyScan1ID, scanName, profileID, scanConfigName, t1, 1)
	framework.InjectScan(timeoutSensor, timeoutScan1ID, scanName, profileID, scanConfigName, t1, 1)

	assert.Eventually(t, func() bool {
		freshness := freshnessByCluster()
		return len(freshness) == 2 &&
			freshness[healthySensor.ClusterID()] == storage.ComplianceOperatorCheckResultV2_CURRENT &&
			freshness[timeoutSensor.ClusterID()] == storage.ComplianceOperatorCheckResultV2_CURRENT
	}, 10*time.Second, 200*time.Millisecond, "both clusters should be CURRENT after cycle 1")

	// Real Compliance Operator scan reruns are minutes to hours apart. Give the report manager's
	// internal ScanWatcher cleanup (handleReadyScan, which consumes the finished watcher's result
	// and removes it from manager_impl's tracking map) a moment to run before cycle 2 reuses the
	// same scan IDs. Reusing a scan ID in the narrow window before that cleanup runs hits a
	// pre-existing race in watcher lifecycle reuse (unrelated to the freshness hooks under test):
	// PushScan against the already-finished-but-not-yet-removed watcher instance silently fails,
	// which this test's unrealistically tight cycle cadence can trigger but a real scan schedule
	// never would.
	time.Sleep(time.Second)

	// =====================================================================
	// Cycle 2 (T2 / T3): the healthy cluster's scan completes normally again at T2, so it stays
	// CURRENT at that timestamp. The timeout cluster's cycle-2 scan is deliberately staggered to
	// a strictly later T3 (= T2 + 1h) and never receives its result, so its scan watcher times
	// out and marks prior results stale with olderThan = T3.
	//
	// T2 and T3 differ on purpose: it makes the test discriminate real per-cluster (scanRef)
	// isolation from accidental scoping by scan name/config (shared across both clusters). Marking
	// stale filters by both scanRef scope AND a LastStartedTime-before-olderThan check
	// (MarkScanChecksStale, datastore_impl.go). With T2 == T3 (as in an earlier version of this
	// test), the timestamp filter alone (T2.Before(T2) == false) would keep the healthy cluster
	// CURRENT even under a hypothetical scan-name-scoped bug, masking the regression this test
	// means to catch. With T3 > T2, that same bug would match the healthy cluster's row (scan
	// name shared) and find T2.Before(T3) == true, wrongly marking it STALE -- so only correct
	// scanRef scoping (which excludes the healthy cluster's row outright, regardless of
	// timestamp) keeps this assertion green.
	// =====================================================================
	// Reuse the same scan IDs as cycle 1: the Scan's ScanRefId is derived from (clusterID,
	// scanName) alone, and the real Compliance Operator updates the same Scan object in place on
	// each run. Minting a new scan ID per cycle would leave two stored rows sharing one
	// ScanRefId, which breaks the check-result pipeline's assumption that a ScanRefId resolves to
	// exactly one scan.
	t2 := t1.Add(time.Hour)
	t3 := t2.Add(time.Hour)
	framework.InjectScan(healthySensor, healthyScan1ID, scanName, profileID, scanConfigName, t2, -1)
	framework.InjectScan(timeoutSensor, timeoutScan1ID, scanName, profileID, scanConfigName, t3, -1)
	// Only the healthy cluster gets its T2 result (same stable result ID, so this upserts in
	// place). Wait for each scan update to land first, same reasoning as cycle 1.
	waitForScanStartTime(healthyScan1ID, t2)
	waitForScanStartTime(timeoutScan1ID, t3)
	framework.InjectCheckResult(healthySensor, healthyResultID, checkID, checkName,
		scanName, scanConfigName, checkName, central.ComplianceOperatorCheckResultV2_PASS, t2)
	assert.Eventually(t, func() bool {
		freshness := freshnessByCluster()
		return freshness[healthySensor.ClusterID()] == storage.ComplianceOperatorCheckResultV2_CURRENT
	}, 10*time.Second, 200*time.Millisecond, "cycle 2 check result should land for the healthy cluster")
	// Update both scan objects (same IDs as above) to expect 1 result; the timeout cluster will
	// never get it and will time out after ROX_COMPLIANCE_SCAN_WATCHER_TIMEOUT (set to 10s above).
	framework.InjectScan(healthySensor, healthyScan1ID, scanName, profileID, scanConfigName, t2, 1)
	framework.InjectScan(timeoutSensor, timeoutScan1ID, scanName, profileID, scanConfigName, t3, 1)

	assert.Eventually(t, func() bool {
		freshness := freshnessByCluster()
		return len(freshness) == 2 &&
			freshness[healthySensor.ClusterID()] == storage.ComplianceOperatorCheckResultV2_CURRENT &&
			freshness[timeoutSensor.ClusterID()] == storage.ComplianceOperatorCheckResultV2_STALE
	}, 30*time.Second, 500*time.Millisecond,
		"timeout cluster should go STALE on watcher timeout while the healthy cluster stays CURRENT")

	// Final sanity check with explicit per-cluster assertions for a clearer failure message. This
	// is the actual point of the PoC: the healthy cluster's result must be untouched by the
	// sibling cluster's scan-watcher timeout, not just "also happen to be CURRENT".
	freshness := freshnessByCluster()
	assert.Equal(t, storage.ComplianceOperatorCheckResultV2_CURRENT,
		freshness[healthySensor.ClusterID()], "healthy cluster's result must remain CURRENT, unaffected by the sibling cluster's timeout")
	assert.Equal(t, storage.ComplianceOperatorCheckResultV2_STALE,
		freshness[timeoutSensor.ClusterID()], "timeout cluster's un-refreshed result must go STALE")
}
