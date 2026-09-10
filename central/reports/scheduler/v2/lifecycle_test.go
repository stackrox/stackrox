package v2

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	configMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	generatorMocks "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator/mocks"
	snapshotMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/postgres"
	pgMocks "github.com/stackrox/rox/pkg/postgres/mocks"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gopkg.in/robfig/cron.v2"
)

type lockResult bool

func (r lockResult) Scan(dest ...interface{}) error {
	*dest[0].(*bool) = bool(r)
	return nil
}

func TestSchedulerRetriesAndRecoversPendingReports(t *testing.T) {
	for name, testCase := range map[string]struct {
		initialError       error
		failSnapshotUpdate bool
	}{
		"lock contention":          {},
		"transient database error": {initialError: errors.New("database unavailable")},
		"snapshot update":          {failSnapshotUpdate: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := pgMocks.NewMockDB(ctrl)
			conn := pgMocks.NewMockPgxPoolConn(ctrl)
			snapshots := snapshotMocks.NewMockDataStore(ctrl)
			configs := configMocks.NewMockDataStore(ctrl)
			generator := generatorMocks.NewMockReportGenerator(ctrl)
			var attempts atomic.Int32
			db.EXPECT().Acquire(gomock.Any()).DoAndReturn(func(context.Context) (*postgres.Conn, error) {
				if attempts.Add(1) == 1 && testCase.initialError != nil {
					return nil, testCase.initialError
				}
				return &postgres.Conn{PgxPoolConn: conn}, nil
			}).AnyTimes()
			conn.EXPECT().QueryRow(gomock.Any(), "SELECT pg_try_advisory_lock($1)", dblock.ReportSchedulerLockID).
				DoAndReturn(func(context.Context, string, ...interface{}) pgx.Row {
					return lockResult(attempts.Load() > 1 || testCase.failSnapshotUpdate)
				}).AnyTimes()
			conn.EXPECT().Release().AnyTimes()
			conn.EXPECT().Exec(gomock.Any(), "SELECT pg_advisory_unlock($1)", dblock.ReportSchedulerLockID).
				Return(pgconn.NewCommandTag("SELECT 1"), nil).AnyTimes()
			pending := &storage.ReportSnapshot{ReportId: "pending", Type: storage.ReportSnapshot_VULNERABILITY,
				ReportStatus: &storage.ReportStatus{ReportRequestType: storage.ReportStatus_VIEW_BASED, RunState: storage.ReportStatus_WAITING}}
			snapshots.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return([]*storage.ReportSnapshot{pending}, nil)
			if testCase.failSnapshotUpdate {
				first := snapshots.EXPECT().UpdateReportSnapshot(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, snap *storage.ReportSnapshot) error {
					assert.Equal(t, "pending", snap.GetReportId())
					return errors.New("temporary update failure")
				})
				snapshots.EXPECT().UpdateReportSnapshot(gomock.Any(), gomock.Any()).After(first).DoAndReturn(func(_ context.Context, snap *storage.ReportSnapshot) error {
					assert.Equal(t, "pending", snap.GetReportId())
					return nil
				})
			} else {
				snapshots.EXPECT().UpdateReportSnapshot(gomock.Any(), gomock.Any()).Return(nil)
			}
			configs.EXPECT().GetReportConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
			processed := make(chan struct{})
			generator.EXPECT().ProcessReportRequest(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *reportGen.ReportRequest) { close(processed) })
			s := newSchedulerImpl(configs, snapshots, nil, nil, generator, nil, nil, runningCron())
			t.Cleanup(s.Stop)
			s.Start(db)
			select {
			case <-processed:
			case <-time.After(7 * time.Second):
				t.Fatal("scheduler did not retry and process the pending report")
			}
			if !testCase.failSnapshotUpdate {
				assert.GreaterOrEqual(t, attempts.Load(), int32(2))
			}
		})
	}
}

func TestSchedulerStopCancelsLockAcquisition(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)
	acquiring := make(chan struct{})
	db.EXPECT().Acquire(gomock.Any()).DoAndReturn(func(ctx context.Context) (*postgres.Conn, error) {
		close(acquiring)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	s := newSchedulerImpl(nil, nil, nil, nil, nil, nil, nil, runningCron())
	started := make(chan struct{})
	go func() { s.Start(db); close(started) }()
	<-acquiring
	s.Stop()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel database lock acquisition")
	}
	require.False(t, s.isStarted.Load())
}

func TestSchedulerStopBeforeStart(t *testing.T) {
	s := newSchedulerImpl(nil, nil, nil, nil, nil, nil, nil, runningCron())
	s.Stop()
	// A stopped scheduler must never touch the database, even on repeated starts.
	s.Start(nil)
	s.Start(nil)
	assert.False(t, s.Ready())
}

func TestSchedulerShutdownWaitsForReportsBeforeUnlock(t *testing.T) {
	t.Setenv("ROX_REPORT_EXECUTION_MAX_CONCURRENCY", "1")
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)
	conn := pgMocks.NewMockPgxPoolConn(ctrl)
	snapshots := snapshotMocks.NewMockDataStore(ctrl)
	configs := configMocks.NewMockDataStore(ctrl)
	generator := generatorMocks.NewMockReportGenerator(ctrl)
	db.EXPECT().Acquire(gomock.Any()).Return(&postgres.Conn{PgxPoolConn: conn}, nil).Times(1)
	conn.EXPECT().QueryRow(gomock.Any(), "SELECT pg_try_advisory_lock($1)", dblock.ReportSchedulerLockID).Return(lockResult(true))
	conn.EXPECT().Release().Times(1)
	var unlocked atomic.Bool
	conn.EXPECT().Exec(gomock.Any(), "SELECT pg_advisory_unlock($1)", dblock.ReportSchedulerLockID).DoAndReturn(
		func(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
			unlocked.Store(true)
			return pgconn.NewCommandTag("SELECT 1"), nil
		})
	initializing := make(chan struct{})
	initialize := make(chan struct{})
	snapshots.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, interface{}) ([]*storage.ReportSnapshot, error) {
			close(initializing)
			<-initialize
			return nil, nil
		})
	configs.EXPECT().GetReportConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	s := newSchedulerImpl(configs, snapshots, nil, nil, generator, nil, nil, runningCron())
	s.Start(db)
	<-initializing
	assert.False(t, s.Ready(), "ownership alone is not readiness before recovery finishes")
	// Start is idempotent even while initialization is running.
	var starts sync.WaitGroup
	for range 10 {
		starts.Go(func() { s.Start(db) })
	}
	starts.Wait()
	close(initialize)
	require.Eventually(t, s.Ready, time.Second, time.Millisecond)
	running := make(chan struct{})
	cancelled := make(chan struct{})
	finish := make(chan struct{})
	generator.EXPECT().ProcessReportRequest(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ *reportGen.ReportRequest) {
			close(running)
			<-ctx.Done()
			close(cancelled)
			<-finish
		})
	q := s.queueByType[storage.ReportSnapshot_VULNERABILITY]
	for _, id := range []string{"first", "second"} {
		q.Enqueue(&reportGen.ReportRequest{ReportSnapshot: &storage.ReportSnapshot{ReportId: id, ReportConfigurationId: id}})
	}
	s.readyForReports.Signal()
	<-running
	stopped := make(chan struct{})
	go func() { s.Stop(); close(stopped) }()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("running report was not cancelled")
	}
	assert.False(t, s.Ready())
	assert.False(t, unlocked.Load(), "ownership must remain until in-flight work exits")
	close(finish)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop blocked behind the report semaphore")
	}
	assert.True(t, unlocked.Load())
	s.Stop()
	s.Start(db)
	assert.False(t, s.Ready())
}

func TestSchedulerStopDuringSuccessfulAcquisition(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)
	conn := pgMocks.NewMockPgxPoolConn(ctrl)
	db.EXPECT().Acquire(gomock.Any()).Return(&postgres.Conn{PgxPoolConn: conn}, nil)
	acquiring := make(chan struct{})
	conn.EXPECT().QueryRow(gomock.Any(), "SELECT pg_try_advisory_lock($1)", dblock.ReportSchedulerLockID).DoAndReturn(
		func(ctx context.Context, _ string, _ ...interface{}) pgx.Row {
			close(acquiring)
			<-ctx.Done()
			return lockResult(true)
		})
	conn.EXPECT().Exec(gomock.Any(), "SELECT pg_advisory_unlock($1)", dblock.ReportSchedulerLockID).Return(pgconn.NewCommandTag("SELECT 1"), nil)
	conn.EXPECT().Release()
	s := newSchedulerImpl(nil, nil, nil, nil, nil, nil, nil, runningCron())
	s.Start(db)
	<-acquiring
	s.Stop()
	assert.False(t, s.Ready())
}

func runningCron() *cron.Cron {
	c := cron.New()
	c.Start()
	return c
}

func TestSchedulerRetriesFailedRecoveryBeforeReadiness(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)
	conn := pgMocks.NewMockPgxPoolConn(ctrl)
	snapshots := snapshotMocks.NewMockDataStore(ctrl)
	configs := configMocks.NewMockDataStore(ctrl)
	db.EXPECT().Acquire(gomock.Any()).Return(&postgres.Conn{PgxPoolConn: conn}, nil)
	conn.EXPECT().QueryRow(gomock.Any(), "SELECT pg_try_advisory_lock($1)", dblock.ReportSchedulerLockID).Return(lockResult(true))
	conn.EXPECT().Exec(gomock.Any(), "SELECT pg_advisory_unlock($1)", dblock.ReportSchedulerLockID).Return(pgconn.NewCommandTag("SELECT 1"), nil)
	conn.EXPECT().Release()
	var recovered atomic.Bool
	first := snapshots.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return(nil, errors.New("temporary read failure"))
	snapshots.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).After(first).DoAndReturn(
		func(context.Context, interface{}) ([]*storage.ReportSnapshot, error) {
			recovered.Store(true)
			return nil, nil
		})
	configs.EXPECT().GetReportConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	s := newSchedulerImpl(configs, snapshots, nil, nil, nil, nil, nil, runningCron())
	t.Cleanup(s.Stop)
	s.Start(db)
	require.Eventually(t, s.Ready, 7*time.Second, time.Millisecond)
	assert.True(t, recovered.Load(), "scheduler became ready without retrying failed recovery")
}
