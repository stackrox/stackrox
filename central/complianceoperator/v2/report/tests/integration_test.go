//go:build sql_integration

package tests

import (
	"context"
	"testing"
	"time"

	framework "github.com/stackrox/rox/central/tests"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReportFiltersStaleResults demonstrates the bug fixed in PR #23560
// (ROX-37684): when a scan watcher times out, DeleteOldResults is skipped,
// leaving stale check results in the database. The report aggregator must
// filter these out by scan timestamp.
func TestReportFiltersStaleResults(t *testing.T) {
	t.Setenv(env.ComplianceScanWatcherTimeout.EnvVar(), "10s")
	t.Setenv(env.ComplianceScanScheduleWatcherTimeout.EnvVar(), "15s")

	c := framework.NewCentral(t)
	c.Start()

	smtp := framework.NewFakeSMTP(t)

	sensor1 := framework.NewFakeSensor(t, "cluster-1")
	sensor1.ConnectTo(c)
	sensor2 := framework.NewFakeSensor(t, "cluster-2")
	sensor2.ConnectTo(c)

	conn := c.APIConn()
	profileID := "xccdf_org.ssgproject.content_profile_cis"
	profileName := "ocp4-cis"
	scanName := profileName
	scanConfigName := "test-scan-config"

	// --- Inject CO resources (compliance info, profiles, rules) via sensor events ---
	for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
		framework.InjectComplianceOperatorInfo(s, "v1.6.0")
	}
	for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
		framework.InjectProfile(s,
			uuid.NewV4().String(), profileName, profileID, "0.1.0", "CIS OpenShift Benchmark", "Platform")
		framework.InjectRule(s, uuid.NewV4().String(),
			"ocp4-api-server-audit-log-maxbackup", "Rule",
			map[string]string{"compliance.openshift.io/profile": profileName})
		framework.InjectRule(s, uuid.NewV4().String(),
			"ocp4-api-server-encryption-provider", "Rule",
			map[string]string{"compliance.openshift.io/profile": profileName})
	}

	profileClient := v2.NewComplianceProfileServiceClient(conn)
	assert.Eventually(t, func() bool {
		resp, err := profileClient.ListComplianceProfiles(context.Background(),
			&v2.ProfilesForClusterRequest{ClusterId: sensor1.ClusterID()})
		return err == nil && len(resp.GetProfiles()) >= 1
	}, 5*time.Second, 100*time.Millisecond)

	// --- Create notifier via API ---
	notifierClient := v1.NewNotifierServiceClient(conn)
	createdNotifier, err := notifierClient.PostNotifier(context.Background(), &storage.Notifier{
		Name:       "test-email",
		Type:       "email",
		UiEndpoint: "https://localhost",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				Server:                   smtp.Addr(),
				Sender:                   "central@test.com",
				DisableTLS:               true,
				AllowUnauthenticatedSmtp: true,
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, createdNotifier.GetId())

	// --- Create scan configuration via API ---
	scanConfigClient := v2.NewComplianceScanConfigurationServiceClient(conn)
	createdScanConfig, err := scanConfigClient.CreateComplianceScanConfiguration(
		context.Background(),
		&v2.ComplianceScanConfiguration{
			ScanName: scanConfigName,
			Clusters: []string{sensor1.ClusterID(), sensor2.ClusterID()},
			ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
				OneTimeScan: true,
				Profiles:    []string{profileName},
				Notifiers: []*v2.NotifierConfiguration{
					{
						NotifierConfig: &v2.NotifierConfiguration_EmailConfig{
							EmailConfig: &v2.EmailNotifierConfiguration{
								NotifierId:   createdNotifier.GetId(),
								MailingLists: []string{"test@test.com"},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, createdScanConfig.GetId())

	resultsClient := v2.NewComplianceResultsServiceClient(conn)

	// =====================================================================
	// First scan cycle (T1): all results arrive, watcher completes normally.
	// This produces a successful report and establishes results in the DB.
	//
	// Event ordering matches the Compliance Operator's real behavior:
	//   1. Scan without check-count (creates watcher, stores scan in DB)
	//   2. Check results (pushed to watcher, stored in DB)
	//   3. Scan with check-count (pushed to existing watcher, sets totalChecks)
	// =====================================================================
	oldScanTime := time.Date(2025, 5, 7, 10, 0, 0, 0, time.UTC)
	newScanTime := time.Date(2025, 5, 7, 11, 0, 0, 0, time.UTC)

	scan1UID := uuid.NewV4().String()
	scan2UID := uuid.NewV4().String()

	// Step 1: inject scans WITHOUT check-count to create watchers and store in DB.
	for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
		uid := scan1UID
		if s == sensor2 {
			uid = scan2UID
		}
		framework.InjectScan(s, uid, scanName, profileID, scanConfigName, oldScanTime, -1)
	}

	// Step 2: poll with probe results until scans are stored in DB.
	assert.Eventually(t, func() bool {
		for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
			framework.InjectCheckResult(s, uuid.NewV4().String(),
				"xccdf_org.ssgproject.content_rule_api_server_audit_log_maxbackup",
				"ocp4-api-server-audit-log-maxbackup",
				scanName, scanConfigName, "ocp4-api-server-audit-log-maxbackup",
				central.ComplianceOperatorCheckResultV2_PASS, oldScanTime)
		}
		resp, err := resultsClient.GetComplianceScanResults(context.Background(), &v2.RawQuery{})
		return err == nil && resp.GetTotalCount() >= 2
	}, 10*time.Second, 500*time.Millisecond)

	// Send the second check result.
	for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
		framework.InjectCheckResult(s, uuid.NewV4().String(),
			"xccdf_org.ssgproject.content_rule_api_server_encryption_provider",
			"ocp4-api-server-encryption-provider",
			scanName, scanConfigName, "ocp4-api-server-encryption-provider",
			central.ComplianceOperatorCheckResultV2_FAIL, oldScanTime)
	}

	assert.Eventually(t, func() bool {
		resp, err := resultsClient.GetComplianceScanResults(context.Background(), &v2.RawQuery{})
		return err == nil && resp.GetTotalCount() >= 4
	}, 10*time.Second, 200*time.Millisecond)

	// Step 3: inject scans WITH check-count=2 to signal watcher completion.
	for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
		uid := scan1UID
		if s == sensor2 {
			uid = scan2UID
		}
		framework.InjectScan(s, uid, scanName, profileID, scanConfigName, oldScanTime, 2)
	}

	// Wait for the first report email.
	assert.Eventually(t, func() bool {
		return len(smtp.Messages()) >= 1
	}, 30*time.Second, 500*time.Millisecond, "first report email should arrive")

	t.Log("First report email received, starting second scan cycle")

	// =====================================================================
	// Second scan cycle (T2): only result-A arrives. result-B is missing.
	// The scan watcher times out (check-count=2 but only 1 result).
	// On timeout, DeleteOldResults is SKIPPED — result-B from T1 stays in DB.
	// The report should filter it out by scan timestamp.
	// =====================================================================
	scan1UID2 := uuid.NewV4().String()
	scan2UID2 := uuid.NewV4().String()

	// Step 1: inject scans WITHOUT check-count.
	for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
		uid := scan1UID2
		if s == sensor2 {
			uid = scan2UID2
		}
		framework.InjectScan(s, uid, scanName, profileID, scanConfigName, newScanTime, -1)
	}

	// Step 2: poll with probe results until scans are stored.
	assert.Eventually(t, func() bool {
		for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
			framework.InjectCheckResult(s, uuid.NewV4().String(),
				"xccdf_org.ssgproject.content_rule_api_server_audit_log_maxbackup",
				"ocp4-api-server-audit-log-maxbackup",
				scanName, scanConfigName, "ocp4-api-server-audit-log-maxbackup",
				central.ComplianceOperatorCheckResultV2_PASS, newScanTime)
		}
		resp, err := resultsClient.GetComplianceScanResults(context.Background(), &v2.RawQuery{})
		return err == nil && resp.GetTotalCount() >= 6
	}, 10*time.Second, 500*time.Millisecond)

	// Do NOT send result-B ("encryption-provider") for T2.
	// Step 3: inject scans WITH check-count=2. The watcher has only 1 result
	// so it will NOT complete — it waits for the missing result and times out.
	// On timeout, DeleteOldResults is skipped → result-B from T1 stays in DB.
	for _, s := range []*framework.FakeSensor{sensor1, sensor2} {
		uid := scan1UID2
		if s == sensor2 {
			uid = scan2UID2
		}
		framework.InjectScan(s, uid, scanName, profileID, scanConfigName, newScanTime, 2)
	}

	// Wait for the second report email (after scan watcher timeout + scan config watcher timeout).
	assert.Eventually(t, func() bool {
		return len(smtp.Messages()) >= 2
	}, 60*time.Second, 500*time.Millisecond, "second report email should arrive after watcher timeout")

	// =====================================================================
	// Assert on the second report: stale result-B should NOT be included.
	// =====================================================================
	require.GreaterOrEqual(t, len(smtp.Messages()), 2, "expected two report emails")

	csvContent, err := smtp.Messages()[1].ExtractCSVFromReport()
	require.NoError(t, err, "failed to extract CSV from report email")
	t.Logf("Second report CSV content:\n%s", csvContent)

	assert.Contains(t, csvContent, "ocp4-api-server-audit-log-maxbackup",
		"report should contain the current check result")
	assert.NotContains(t, csvContent, "ocp4-api-server-encryption-provider",
		"report should NOT contain the stale check result from the older scan")
}
