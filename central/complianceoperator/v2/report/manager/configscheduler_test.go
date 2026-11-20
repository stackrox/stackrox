package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// loopNow is the frozen time loop tests run at: Friday, November 20 2026,
// 12:00 UTC.
var loopNow = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

// loopTestConfig returns a daily scan configuration firing at 11:00 UTC (one
// hour before the frozen loop time), with a notifier configured.
func loopTestConfig(id string) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:             id,
		ScanConfigName: id,
		Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
			{ClusterId: "cluster-1"},
		},
		Notifiers: []*storage.NotifierConfiguration{
			{
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "notifier-1",
						MailingLists: []string{"test@test.com"},
					},
				},
			},
		},
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         11,
			Minute:       0,
		},
	}
}

func loopSnapshotOfType(requestType storage.ComplianceOperatorReportStatus_RunMethod, startedAgo time.Duration) *storage.ComplianceOperatorReportSnapshotV2 {
	started := loopNow.Add(-startedAgo)
	return &storage.ComplianceOperatorReportSnapshotV2{
		ReportId:            "report-1",
		ScanConfigurationId: "config-1",
		ReportStatus: &storage.ComplianceOperatorReportStatus{
			ReportRequestType: requestType,
			StartedAt:         protocompat.ConvertTimeToTimestampOrNil(&started),
		},
	}
}

// loopClock makes the scheduler loop deterministic: sleep advances the clock
// and records the waited duration, and the loop is ended by returning false
// after stopAfter sleeps (simulating a stop request during a sleep).
type loopClock struct {
	t          time.Time
	sleeps     []time.Duration
	stopAfter  int
	sleepCount int
}

func (c *loopClock) now() time.Time { return c.t }

func (c *loopClock) sleep(_ <-chan struct{}, d time.Duration) bool {
	c.sleeps = append(c.sleeps, d)
	c.t = c.t.Add(d)
	c.sleepCount++
	return c.sleepCount < c.stopAfter
}

// newTestScheduler builds a configScheduler on freshly created mocks. The
// scan configuration (or error) served by the datastore and the snapshots of
// previous processes are controlled through the returned setters.
func newTestScheduler(t *testing.T) (*configScheduler, *loopClock, *func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error), *[]*storage.ComplianceOperatorReportSnapshotV2, *[]*storage.ComplianceOperatorScanConfigurationV2, *func() error) {
	t.Helper()
	ctrl := gomock.NewController(t)
	scanConfigDS := scanConfigurationDS.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)

	clock := &loopClock{t: loopNow, stopAfter: 100}
	getConfig := func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
		return loopTestConfig("config-1"), true, nil
	}
	var snapshots []*storage.ComplianceOperatorReportSnapshotV2
	var submitted []*storage.ComplianceOperatorScanConfigurationV2
	submitErr := func() error { return nil }

	scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), gomock.Eq("config-1")).AnyTimes().
		DoAndReturn(func(context.Context, string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
			return getConfig()
		})
	snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, _ *v1.Query) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
			ret := make([]*storage.ComplianceOperatorReportSnapshotV2, len(snapshots))
			copy(ret, snapshots)
			return ret, nil
		})

	s := &configScheduler{
		id:           "config-1",
		ctx:          context.Background(),
		scanConfigDS: scanConfigDS,
		snapshotDS:   snapshotDS,
		submit: func(scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
			if err := submitErr(); err != nil {
				return err
			}
			submitted = append(submitted, scanConfig)
			return nil
		},
		delay:            time.Hour,
		recheckInterval:  15 * time.Minute,
		retryInterval:    30 * time.Second,
		missingThreshold: 3,
		now:              clock.now,
		sleep:            clock.sleep,
		stopper:          concurrency.NewStopper(),
	}
	return s, clock, &getConfig, &snapshots, &submitted, &submitErr
}

func TestConfigSchedulerRun(t *testing.T) {
	tests := map[string]struct {
		config        func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error)
		snapshots     []*storage.ComplianceOperatorReportSnapshotV2
		stopAfter     int // number of sleeps before the loop is stopped
		expectSubmits int
		assertExtra   func(t *testing.T, clock *loopClock)
	}{
		"due report is submitted once and the loop idles covered": {
			config: func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				return loopTestConfig("config-1"), true, nil
			},
			stopAfter:     3,
			expectSubmits: 1,
			assertExtra: func(t *testing.T, clock *loopClock) {
				// After the submit the loop idles on the recheck cadence.
				require.NotEmpty(t, clock.sleeps)
				assert.Equal(t, 15*time.Minute, clock.sleeps[len(clock.sleeps)-1])
			},
		},
		"report not yet due sleeps until due then submits": {
			config: func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				sc := loopTestConfig("config-1")
				sc.Schedule.Minute = 30 // fires 11:30, due 12:30
				return sc, true, nil
			},
			stopAfter:     2,
			expectSubmits: 1,
			assertExtra: func(t *testing.T, clock *loopClock) {
				require.NotEmpty(t, clock.sleeps)
				assert.Equal(t, 30*time.Minute, clock.sleeps[0], "the loop should sleep until the report is due")
			},
		},
		"scheduled snapshot from a previous process covers the fire": {
			config: func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				return loopTestConfig("config-1"), true, nil
			},
			snapshots: []*storage.ComplianceOperatorReportSnapshotV2{loopSnapshotOfType(storage.ComplianceOperatorReportStatus_SCHEDULED, 15*time.Minute)},
			stopAfter: 2,
		},
		"on-demand snapshot does not cover the fire": {
			config: func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				return loopTestConfig("config-1"), true, nil
			},
			snapshots:     []*storage.ComplianceOperatorReportSnapshotV2{loopSnapshotOfType(storage.ComplianceOperatorReportStatus_ON_DEMAND, 15*time.Minute)},
			stopAfter:     2,
			expectSubmits: 1,
		},
		"one-time config is reported once after creation plus delay": {
			config: func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				sc := loopTestConfig("config-1")
				sc.Schedule = nil
				sc.OneTimeScan = true
				created := loopNow.Add(-2 * time.Hour)
				sc.CreatedTime = protocompat.ConvertTimeToTimestampOrNil(&created)
				return sc, true, nil
			},
			stopAfter:     3,
			expectSubmits: 1,
		},
		"config without notifiers idles": {
			config: func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				sc := loopTestConfig("config-1")
				sc.Notifiers = nil
				return sc, true, nil
			},
			stopAfter: 2,
		},
		"config younger than the last fire idles until a fire happens": {
			config: func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				sc := loopTestConfig("config-1")
				created := loopNow.Add(-30 * time.Minute)
				sc.CreatedTime = protocompat.ConvertTimeToTimestampOrNil(&created)
				return sc, true, nil
			},
			stopAfter: 2,
			assertExtra: func(t *testing.T, clock *loopClock) {
				require.NotEmpty(t, clock.sleeps)
				assert.Equal(t, 15*time.Minute, clock.sleeps[0], "no fire to wait for: recheck cadence")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, clock, getConfig, snapshots, submitted, _ := newTestScheduler(t)
			*getConfig = tc.config
			*snapshots = tc.snapshots
			clock.stopAfter = tc.stopAfter

			s.Run()

			assert.Len(t, *submitted, tc.expectSubmits)
			if tc.assertExtra != nil {
				tc.assertExtra(t, clock)
			}
			assert.NoError(t, s.stopper.Client().Stopped().Err(), "a stop during sleep carries no error")
		})
	}
}

// TestConfigSchedulerRunRetryOnError covers the transient-read-error retry
// separately, since it needs the served configuration to change mid-loop.
func TestConfigSchedulerRunRetryOnError(t *testing.T) {
	s, clock, getConfig, _, submitted, _ := newTestScheduler(t)
	fail := true
	*getConfig = func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
		if fail {
			fail = false
			return nil, false, errors.New("db down")
		}
		return loopTestConfig("config-1"), true, nil
	}
	clock.stopAfter = 3

	s.Run()

	assert.Len(t, *submitted, 1, "the report is submitted after the read error clears")
	require.NotEmpty(t, clock.sleeps)
	assert.Equal(t, s.retryInterval, clock.sleeps[0], "the read error is backed off")
}

// TestConfigSchedulerRunRetrySubmitFailure verifies that a failed submission
// does not record the fire, so it is retried.
func TestConfigSchedulerRunRetrySubmitFailure(t *testing.T) {
	s, clock, _, _, submitted, submitErr := newTestScheduler(t)
	fail := true
	*submitErr = func() error {
		if fail {
			fail = false
			return assert.AnError
		}
		return nil
	}
	clock.stopAfter = 3

	s.Run()

	assert.Len(t, *submitted, 1, "the fire is retried after the failed submission")
	require.NotEmpty(t, clock.sleeps)
	assert.Equal(t, s.retryInterval, clock.sleeps[0], "the failed submission is backed off")
}

// TestConfigSchedulerRunMissingConfig verifies that a configuration that
// stays missing beyond the threshold exits the loop with errScanConfigGone,
// so the manager deregisters the scheduler instead of respawning it.
func TestConfigSchedulerRunMissingConfig(t *testing.T) {
	s, clock, getConfig, _, submitted, _ := newTestScheduler(t)
	*getConfig = func() (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
		return nil, false, nil
	}
	clock.stopAfter = 100 // the loop must exit on its own, not via the sleep cutoff

	s.Run()

	assert.Empty(t, *submitted)
	assert.ErrorIs(t, s.stopper.Client().Stopped().Err(), errScanConfigGone)
	require.NotEmpty(t, clock.sleeps)
	assert.Len(t, clock.sleeps, s.missingThreshold-1, "one retry sleep per missing read before the last one exits")
}
