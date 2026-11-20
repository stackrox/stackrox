package settle

import (
	"context"
	"fmt"
	"maps"
	"testing"
	"time"

	"github.com/pkg/errors"
	integrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// frozenNow is the reference time all settle tests run at:
// Friday, November 20 2026, 12:00 UTC.
var frozenNow = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

// fakeClock makes the settle loop deterministic: sleep advances the clock
// instead of blocking.
type fakeClock struct {
	t      time.Time
	sleeps []time.Duration
}

func (c *fakeClock) now() time.Time { return c.t }

func (c *fakeClock) sleep(_ context.Context, d time.Duration) bool {
	c.sleeps = append(c.sleeps, d)
	c.t = c.t.Add(d)
	return true
}

type fakeClusterSource struct {
	clusters map[string]*storage.Cluster
}

func (f *fakeClusterSource) GetCluster(_ context.Context, id string) (*storage.Cluster, bool, error) {
	cluster, found := f.clusters[id]
	return cluster, found, nil
}

func healthyCluster(id string) *storage.Cluster {
	return &storage.Cluster{
		Id:           id,
		HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
	}
}

func healthyIntegration() []*storage.ComplianceIntegration {
	return []*storage.ComplianceIntegration{
		{
			OperatorInstalled: true,
			Version:           "v1.6.0",
		},
	}
}

// newTestSettler builds a settler on mutable fakes: the scan datastore and the
// compliance integration datastore are backed by functions the test case can
// swap, which keeps gomock call ordering out of the way.
func newTestSettler(t *testing.T) (*Settler, *fakeClock, *func() ([]*storage.ComplianceOperatorScanV2, error), *func(string) ([]*storage.ComplianceIntegration, error), *fakeClusterSource) {
	t.Helper()
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	integrationDS := integrationMocks.NewMockDataStore(ctrl)
	clock := &fakeClock{t: frozenNow}
	clusters := &fakeClusterSource{clusters: make(map[string]*storage.Cluster)}

	scansResp := func() ([]*storage.ComplianceOperatorScanV2, error) { return nil, nil }
	integrationResp := func(string) ([]*storage.ComplianceIntegration, error) { return healthyIntegration(), nil }

	scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(context.Context, any) ([]*storage.ComplianceOperatorScanV2, error) {
			return scansResp()
		})
	integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, clusterID string) ([]*storage.ComplianceIntegration, error) {
			return integrationResp(clusterID)
		})

	settler := &Settler{
		scanDS:        scanDS,
		integrationDS: integrationDS,
		clusters:      clusters,
		timeout:       3 * time.Minute,
		pollInterval:  time.Minute,
		grace:         time.Hour,
		now:           clock.now,
		sleep:         clock.sleep,
	}
	return settler, clock, &scansResp, &integrationResp, clusters
}

// getScanConfig returns a daily scan configuration firing at 12:00 UTC. The
// tests run at exactly the fire time, i.e. within the 1h grace period, so the
// audit cutoff is the previous day's fire.
func getScanConfig() *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "ocp4-bsi-node",
		Id:             "scan-config-id",
		Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
			{ClusterId: "cluster-1"},
		},
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         12,
			Minute:       0,
		},
	}
}

// neverStarted is the sentinel for scanWithPhase's startedAgo parameter: the
// scan has no LastStartedTime.
const neverStarted time.Duration = -1

func scanWithPhase(id, phase, result string, startedAgo time.Duration) *storage.ComplianceOperatorScanV2 {
	scan := &storage.ComplianceOperatorScanV2{
		Id:        "id-" + id,
		ClusterId: "cluster-1",
		ScanName:  id,
		ScanRefId: "ref-" + id,
		Status:    &storage.ScanStatus{Phase: phase, Result: result},
	}
	if startedAgo >= 0 {
		start := frozenNow.Add(-startedAgo)
		scan.LastStartedTime = protocompat.ConvertTimeToTimestampOrNil(&start)
	}
	return scan
}

func TestSettle(t *testing.T) {
	inFlight := func(id string) *storage.ComplianceOperatorScanV2 {
		return scanWithPhase(id, "RUNNING", "", 10*time.Minute)
	}
	done := func(scan *storage.ComplianceOperatorScanV2) *storage.ComplianceOperatorScanV2 {
		ret := scan.CloneVT()
		ret.Status = &storage.ScanStatus{Phase: "DONE", Result: "COMPLIANT"}
		return ret
	}

	tests := map[string]struct {
		scansByCall [][]*storage.ComplianceOperatorScanV2 // scans returned per SearchScans call; the last entry repeats
		errOnCall   int                                   // 1-based call number returning an error; 0 disables
		expectErr   string
		assertRes   func(t *testing.T, res *SettleResult, clock *fakeClock)
	}{
		"no scans at all": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{nil},
			assertRes: func(t *testing.T, res *SettleResult, clock *fakeClock) {
				assert.Empty(t, res.Running)
				assert.Empty(t, res.Unfinished)
				assert.Empty(t, res.Removed)
				assert.Empty(t, clock.sleeps)
			},
		},
		"no scans in flight": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{
				{done(inFlight("scan-1"))},
			},
			assertRes: func(t *testing.T, res *SettleResult, clock *fakeClock) {
				assert.Empty(t, res.Running)
				assert.Empty(t, clock.sleeps)
			},
		},
		"scan without an observed phase is not waited on": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{
				{scanWithPhase("scan-1", "", "", neverStarted)},
			},
			assertRes: func(t *testing.T, res *SettleResult, clock *fakeClock) {
				assert.Empty(t, res.Running)
				assert.Empty(t, clock.sleeps)
			},
		},
		"running scan finishes on the next poll": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{
				{inFlight("scan-1")},
				{done(inFlight("scan-1"))},
			},
			assertRes: func(t *testing.T, res *SettleResult, clock *fakeClock) {
				assert.Len(t, res.Running, 1)
				assert.Empty(t, res.Unfinished)
				assert.Empty(t, res.Removed)
				assert.Len(t, clock.sleeps, 1)
			},
		},
		"multiple polls until the scans finish": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{
				{inFlight("scan-1"), inFlight("scan-2")},
				{inFlight("scan-1"), inFlight("scan-2")},
				{done(inFlight("scan-1")), inFlight("scan-2")},
				{done(inFlight("scan-1")), done(inFlight("scan-2"))},
			},
			assertRes: func(t *testing.T, res *SettleResult, clock *fakeClock) {
				assert.Len(t, res.Running, 2)
				assert.Empty(t, res.Unfinished)
				assert.Len(t, clock.sleeps, 3)
			},
		},
		"running scans are unfinished after the timeout": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{
				{inFlight("scan-1"), inFlight("scan-2")},
			},
			assertRes: func(t *testing.T, res *SettleResult, clock *fakeClock) {
				assert.Len(t, res.Running, 2)
				require.Len(t, res.Unfinished, 2)
				assert.Equal(t, "scan-1", res.Unfinished[0].GetScanName())
				assert.Equal(t, "scan-2", res.Unfinished[1].GetScanName())
				// timeout 3m, poll 1m: three polls, then the deadline.
				assert.Len(t, clock.sleeps, 3)
			},
		},
		"scan removed while waited on": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{
				{inFlight("scan-1")},
				{},
			},
			assertRes: func(t *testing.T, res *SettleResult, clock *fakeClock) {
				assert.Len(t, res.Running, 1)
				assert.Empty(t, res.Unfinished)
				require.Len(t, res.Removed, 1)
				assert.Equal(t, "scan-1", res.Removed[0].GetScanName())
			},
		},
		"datastore error while polling fails the settle": {
			scansByCall: [][]*storage.ComplianceOperatorScanV2{
				{inFlight("scan-1")},
			},
			errOnCall: 2,
			expectErr: "unable to retrieve scans",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			settler, clock, scansResp, _, _ := newTestSettler(t)
			call := 0
			*scansResp = func() ([]*storage.ComplianceOperatorScanV2, error) {
				call++
				if tc.errOnCall > 0 && call == tc.errOnCall {
					return nil, errors.New("datastore boom")
				}
				idx := min(call-1, len(tc.scansByCall)-1)
				return tc.scansByCall[idx], nil
			}

			res, err := settler.Settle(context.Background(), getScanConfig())
			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}
			require.NoError(t, err)
			tc.assertRes(t, res, clock)
		})
	}
}

func TestSettleContextCancelled(t *testing.T) {
	settler, _, scansResp, _, _ := newTestSettler(t)
	*scansResp = func() ([]*storage.ComplianceOperatorScanV2, error) {
		return []*storage.ComplianceOperatorScanV2{scanWithPhase("scan-1", "RUNNING", "", 10*time.Minute)}, nil
	}
	settler.sleep = func(ctx context.Context, _ time.Duration) bool {
		<-ctx.Done()
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := settler.Settle(ctx, getScanConfig())
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAudit(t *testing.T) {
	freshAgo := 30 * time.Minute // today 11:30, after yesterday's fire
	staleAgo := 48 * time.Hour   // two days ago, before yesterday's fire
	inFlightAgo := 10 * time.Minute

	fresh := func(name string) *storage.ComplianceOperatorScanV2 {
		return scanWithPhase(name, "DONE", "COMPLIANT", freshAgo)
	}
	stale := func(name string) *storage.ComplianceOperatorScanV2 {
		return scanWithPhase(name, "DONE", "COMPLIANT", staleAgo)
	}
	neverRan := func(name string) *storage.ComplianceOperatorScanV2 {
		return scanWithPhase(name, "DONE", "COMPLIANT", neverStarted)
	}

	tests := map[string]struct {
		scanConfig    *storage.ComplianceOperatorScanConfigurationV2
		scans         []*storage.ComplianceOperatorScanV2
		settleRes     *SettleResult
		clusters      map[string]*storage.Cluster
		integrationFn func(string) ([]*storage.ComplianceIntegration, error)
		assertRes     func(t *testing.T, res *AuditResult)
	}{
		"all scans fresh": {
			scanConfig: getScanConfig(),
			scans:      []*storage.ComplianceOperatorScanV2{fresh("ocp4-master"), fresh("ocp4-worker")},
			settleRes:  &SettleResult{},
			assertRes: func(t *testing.T, res *AuditResult) {
				assert.Empty(t, res.FailedClusters)
				require.Len(t, res.Scans, 2)
			},
		},
		"overdue scan flags the cluster but keeps its results": {
			scanConfig: getScanConfig(),
			scans:      []*storage.ComplianceOperatorScanV2{fresh("ocp4-infra"), stale("ocp4-master")},
			settleRes:  &SettleResult{},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				fc := res.FailedClusters["cluster-1"]
				assert.Empty(t, fc.FailedScans, "overdue scans are not excluded from the report content")
				assert.True(t, fc.IncludeResults, "overdue-only findings keep the cluster's results")
				assert.Contains(t, fc.Reasons[0], "ocp4-master is overdue")
			},
		},
		"unfinished scan is a failed scan excluded from the content": {
			scanConfig: getScanConfig(),
			scans:      []*storage.ComplianceOperatorScanV2{fresh("ocp4-master"), scanWithPhase("ocp4-infra", "RUNNING", "", inFlightAgo)},
			settleRes: &SettleResult{
				Unfinished: []*storage.ComplianceOperatorScanV2{scanWithPhase("ocp4-infra", "RUNNING", "", inFlightAgo)},
			},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				fc := res.FailedClusters["cluster-1"]
				require.Len(t, fc.FailedScans, 1)
				assert.Equal(t, "ocp4-infra", fc.FailedScans[0].GetScanName())
				assert.False(t, fc.IncludeResults)
				// The unfinished scan is not also reported as overdue.
				assert.Len(t, fc.Reasons, 1)
				assert.Contains(t, fc.Reasons[0], "Timeout waiting for scan ocp4-infra")
			},
		},
		"removed scan is flagged with the removal reason": {
			scanConfig: getScanConfig(),
			scans:      []*storage.ComplianceOperatorScanV2{fresh("ocp4-master")},
			settleRes: &SettleResult{
				Removed: []*storage.ComplianceOperatorScanV2{stale("ocp4-infra")},
			},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				fc := res.FailedClusters["cluster-1"]
				assert.Contains(t, fc.Reasons[0], "ocp4-infra was removed")
				assert.True(t, fc.IncludeResults, "the surviving scans' results stay in the report")
			},
		},
		"scan that never started is flagged when there is an expectation": {
			scanConfig: getScanConfig(),
			scans:      []*storage.ComplianceOperatorScanV2{neverRan("ocp4-master")},
			settleRes:  &SettleResult{},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				assert.Contains(t, res.FailedClusters["cluster-1"].Reasons[0], "ocp4-master is overdue: it has never started")
			},
		},
		"no schedule means no overdue classification": {
			scanConfig: func() *storage.ComplianceOperatorScanConfigurationV2 {
				sc := getScanConfig()
				sc.Schedule = nil
				return sc
			}(),
			scans:     []*storage.ComplianceOperatorScanV2{stale("ocp4-master")},
			settleRes: &SettleResult{},
			assertRes: func(t *testing.T, res *AuditResult) {
				assert.Empty(t, res.FailedClusters)
			},
		},
		"configuration younger than the last fire means no expectation": {
			scanConfig: func() *storage.ComplianceOperatorScanConfigurationV2 {
				sc := getScanConfig()
				// Fire daily at 11:00: the last fire was an hour before the
				// (frozen) report time. The configuration was created at
				// 11:30, after that fire, so no scheduled fire has happened
				// since it was created and nothing can be overdue yet.
				sc.Schedule.Hour = 11
				created := frozenNow.Add(-30 * time.Minute)
				sc.CreatedTime = protocompat.ConvertTimeToTimestampOrNil(&created)
				return sc
			}(),
			scans:     []*storage.ComplianceOperatorScanV2{stale("ocp4-master")},
			settleRes: &SettleResult{},
			assertRes: func(t *testing.T, res *AuditResult) {
				assert.Empty(t, res.FailedClusters)
			},
		},
		"within grace of the latest fire the previous fire is the cutoff": {
			scanConfig: getScanConfig(),
			scans: []*storage.ComplianceOperatorScanV2{
				scanWithPhase("ocp4-master", "DONE", "COMPLIANT", 30*time.Hour), // before yesterday's fire
				scanWithPhase("ocp4-worker", "DONE", "COMPLIANT", 20*time.Hour), // after yesterday's fire
			},
			settleRes: &SettleResult{},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				fc := res.FailedClusters["cluster-1"]
				assert.Len(t, fc.Reasons, 1)
				assert.Contains(t, fc.Reasons[0], "ocp4-master is overdue")
			},
		},
		"cluster without any scan is flagged with derived reasons": {
			scanConfig: func() *storage.ComplianceOperatorScanConfigurationV2 {
				sc := getScanConfig()
				sc.Clusters = append(sc.GetClusters(), &storage.ComplianceOperatorScanConfigurationV2_Cluster{ClusterId: "cluster-2"})
				return sc
			}(),
			scans:     []*storage.ComplianceOperatorScanV2{fresh("ocp4-master")},
			settleRes: &SettleResult{},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				fc := res.FailedClusters["cluster-2"]
				assert.Contains(t, fc.Reasons, report.NO_SCANS_OF_CONFIG_FOUND)
				assert.False(t, fc.IncludeResults)
			},
		},
		"cluster without any scan gets the sensor disconnected reason": {
			scanConfig: func() *storage.ComplianceOperatorScanConfigurationV2 {
				sc := getScanConfig()
				sc.Clusters = append(sc.GetClusters(), &storage.ComplianceOperatorScanConfigurationV2_Cluster{ClusterId: "cluster-2"})
				return sc
			}(),
			scans:     []*storage.ComplianceOperatorScanV2{fresh("ocp4-master")},
			settleRes: &SettleResult{},
			clusters: map[string]*storage.Cluster{
				"cluster-2": {
					Id:           "cluster-2",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY},
				},
			},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				assert.Contains(t, res.FailedClusters["cluster-2"].Reasons, report.SENSOR_DISCONNECTED)
			},
		},
		"cluster without any scan gets the compliance operator reason": {
			scanConfig: func() *storage.ComplianceOperatorScanConfigurationV2 {
				sc := getScanConfig()
				sc.Clusters = append(sc.GetClusters(), &storage.ComplianceOperatorScanConfigurationV2_Cluster{ClusterId: "cluster-2"})
				return sc
			}(),
			scans:     []*storage.ComplianceOperatorScanV2{fresh("ocp4-master")},
			settleRes: &SettleResult{},
			integrationFn: func(clusterID string) ([]*storage.ComplianceIntegration, error) {
				if clusterID == "cluster-2" {
					return []*storage.ComplianceIntegration{{OperatorInstalled: false}}, nil
				}
				return healthyIntegration(), nil
			},
			assertRes: func(t *testing.T, res *AuditResult) {
				require.Len(t, res.FailedClusters, 1)
				assert.Contains(t, res.FailedClusters["cluster-2"].Reasons, report.COMPLIANCE_NOT_INSTALLED)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			settler, _, scansResp, integrationResp, clusters := newTestSettler(t)
			*scansResp = func() ([]*storage.ComplianceOperatorScanV2, error) {
				return tc.scans, nil
			}
			if tc.integrationFn != nil {
				*integrationResp = tc.integrationFn
			}
			maps.Copy(clusters.clusters, tc.clusters)

			res, err := settler.Audit(context.Background(), tc.scanConfig, tc.settleRes)
			require.NoError(t, err)
			tc.assertRes(t, res)
		})
	}
}

// TestReportDue covers the scheduler-facing schedule computation: which fire
// a scheduled report corresponds to, and when it is due.
func TestReportDue(t *testing.T) {
	// Friday, November 20 2026, 12:00 UTC.
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	delay := time.Hour

	tests := map[string]struct {
		scanConfig *storage.ComplianceOperatorScanConfigurationV2
		expectFire time.Time
		expectDue  time.Time
		expectOK   bool
	}{
		"daily configuration due after the fire plus delay": {
			scanConfig: &storage.ComplianceOperatorScanConfigurationV2{
				Schedule: &storage.Schedule{IntervalType: storage.Schedule_DAILY, Hour: 11, Minute: 0},
			},
			expectFire: time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC),
			expectDue:  time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC),
			expectOK:   true,
		},
		"nil schedule means no cron-based report": {
			scanConfig: &storage.ComplianceOperatorScanConfigurationV2{},
		},
		"configuration created after the last fire means no report yet": {
			scanConfig: func() *storage.ComplianceOperatorScanConfigurationV2 {
				sc := &storage.ComplianceOperatorScanConfigurationV2{
					Schedule: &storage.Schedule{IntervalType: storage.Schedule_DAILY, Hour: 11, Minute: 0},
				}
				created := time.Date(2026, 11, 20, 11, 30, 0, 0, time.UTC)
				sc.CreatedTime = protocompat.ConvertTimeToTimestampOrNil(&created)
				return sc
			}(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fire, due, ok := ReportDue(tc.scanConfig, now, delay)
			assert.Equal(t, tc.expectOK, ok, "ok")
			assert.True(t, tc.expectFire.Equal(fire), fmt.Sprintf("fire: got %v, want %v", fire, tc.expectFire))
			assert.True(t, tc.expectDue.Equal(due), fmt.Sprintf("due: got %v, want %v", due, tc.expectDue))
		})
	}
}

func TestLastTwoFires(t *testing.T) {
	// Wednesday, November 18 2026, 10:30 UTC.
	now := time.Date(2026, 11, 18, 10, 30, 0, 0, time.UTC)

	tests := map[string]struct {
		schedule   *storage.Schedule
		now        time.Time
		expectLast time.Time
		expectPrev time.Time
		expectOK   bool
	}{
		"nil schedule": {
			schedule: nil,
			expectOK: false,
		},
		"unset interval type": {
			schedule: &storage.Schedule{IntervalType: storage.Schedule_UNSET, Hour: 1, Minute: 5},
			expectOK: false,
		},
		"weekly schedule without days": {
			schedule: &storage.Schedule{IntervalType: storage.Schedule_WEEKLY, Hour: 1, Minute: 5},
			expectOK: false,
		},
		"daily before today's fire": {
			schedule:   &storage.Schedule{IntervalType: storage.Schedule_DAILY, Hour: 1, Minute: 5},
			now:        now,
			expectLast: time.Date(2026, 11, 18, 1, 5, 0, 0, time.UTC),
			expectPrev: time.Date(2026, 11, 17, 1, 5, 0, 0, time.UTC),
			expectOK:   true,
		},
		"daily after today's fire": {
			schedule:   &storage.Schedule{IntervalType: storage.Schedule_DAILY, Hour: 23, Minute: 5},
			now:        now,
			expectLast: time.Date(2026, 11, 17, 23, 5, 0, 0, time.UTC),
			expectPrev: time.Date(2026, 11, 16, 23, 5, 0, 0, time.UTC),
			expectOK:   true,
		},
		"daily exactly at the fire": {
			schedule:   &storage.Schedule{IntervalType: storage.Schedule_DAILY, Hour: 10, Minute: 30},
			now:        now,
			expectLast: now,
			expectPrev: time.Date(2026, 11, 17, 10, 30, 0, 0, time.UTC),
			expectOK:   true,
		},
		"weekly on a single day that already passed this week": {
			// 0 = Sunday. Now is Wednesday; the last Sunday fire was Nov 15.
			schedule: &storage.Schedule{
				IntervalType: storage.Schedule_WEEKLY,
				Hour:         2, Minute: 0,
				Interval: &storage.Schedule_Weekly{Weekly: &storage.Schedule_WeeklyInterval{Day: 0}},
			},
			now:        now,
			expectLast: time.Date(2026, 11, 15, 2, 0, 0, 0, time.UTC),
			expectPrev: time.Date(2026, 11, 8, 2, 0, 0, 0, time.UTC),
			expectOK:   true,
		},
		"weekly on multiple days": {
			// 1 = Monday, 4 = Thursday. Now is Wednesday: the last fire was
			// Monday, the one before that Thursday of the previous week.
			schedule: &storage.Schedule{
				IntervalType: storage.Schedule_WEEKLY,
				Hour:         0, Minute: 15,
				Interval: &storage.Schedule_DaysOfWeek_{DaysOfWeek: &storage.Schedule_DaysOfWeek{Days: []int32{1, 4}}},
			},
			now:        now,
			expectLast: time.Date(2026, 11, 16, 0, 15, 0, 0, time.UTC),
			expectPrev: time.Date(2026, 11, 12, 0, 15, 0, 0, time.UTC),
			expectOK:   true,
		},
		"monthly on a day that already passed": {
			schedule: &storage.Schedule{
				IntervalType: storage.Schedule_MONTHLY,
				Hour:         3, Minute: 0,
				Interval: &storage.Schedule_DaysOfMonth_{DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: []int32{1, 15}}},
			},
			now:        now,
			expectLast: time.Date(2026, 11, 15, 3, 0, 0, 0, time.UTC),
			expectPrev: time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC),
			expectOK:   true,
		},
		"monthly day-of-month absent from short months": {
			// Day 31: November has 30 days, so from Nov 18 the last fire is
			// Oct 31; September has no 31st, so the one before is Aug 31.
			schedule: &storage.Schedule{
				IntervalType: storage.Schedule_MONTHLY,
				Hour:         0, Minute: 0,
				Interval: &storage.Schedule_DaysOfMonth_{DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: []int32{31}}},
			},
			now:        now,
			expectLast: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
			expectPrev: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
			expectOK:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			last, prev, ok := lastTwoFires(tc.schedule, tc.now)
			assert.Equal(t, tc.expectOK, ok, "ok")
			assert.True(t, tc.expectLast.Equal(last), fmt.Sprintf("last: got %v, want %v", last, tc.expectLast))
			assert.True(t, tc.expectPrev.Equal(prev), fmt.Sprintf("prev: got %v, want %v", prev, tc.expectPrev))
		})
	}
}
