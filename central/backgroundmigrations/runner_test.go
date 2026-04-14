package backgroundmigrations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stackrox/rox/pkg/postgres"
	pgMocks "github.com/stackrox/rox/pkg/postgres/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeTryLockFunc returns a tryAcquireLockFunc that always succeeds immediately.
func fakeTryLockFunc() func(ctx context.Context, db postgres.DB, lockID int64) (bool, func(), error) {
	return func(_ context.Context, _ postgres.DB, _ int64) (bool, func(), error) {
		return true, func() {}, nil
	}
}

// neverAcquireLockFunc returns a tryAcquireLockFunc that never acquires until context is cancelled.
func neverAcquireLockFunc() func(ctx context.Context, db postgres.DB, lockID int64) (bool, func(), error) {
	return func(_ context.Context, _ postgres.DB, _ int64) (bool, func(), error) {
		return false, nil, nil
	}
}

const testTimeout = 1 * time.Second

// fakeRow implements pgx.Row for returning a seqnum from QueryRow.
type fakeRow struct {
	seqNum int32
	err    error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) > 0 {
		if p, ok := dest[0].(*int32); ok {
			*p = r.seqNum
		}
	}
	return nil
}

var _ pgx.Row = &fakeRow{}

type noopRolloutChecker struct{}

func (n *noopRolloutChecker) WaitForRolloutComplete(_ context.Context) error { return nil }

// blockingRolloutChecker blocks until context is cancelled — used to test stop signal propagation.
type blockingRolloutChecker struct{}

func (c *blockingRolloutChecker) WaitForRolloutComplete(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

var _ RolloutChecker = &noopRolloutChecker{}
var _ RolloutChecker = &k8sRolloutChecker{}
var _ RolloutChecker = &blockingRolloutChecker{}

func resetRegistry() {
	migrations = make(map[int]BackgroundMigration)
}

func newTestRunner(db postgres.DB, rolloutChecker RolloutChecker, currentSeqNum int) *Runner {
	r := NewRunner(db, rolloutChecker)
	r.currentBgSeqNumFunc = func() int { return currentSeqNum }
	r.tryAcquireLockFunc = fakeTryLockFunc()
	r.lockRetryInterval = 10 * time.Millisecond
	return r
}

// requireStoppedWithin starts the runner and fails the test if it doesn't stop within the timeout.
func requireStoppedWithin(t *testing.T, runner *Runner, timeout time.Duration) {
	t.Helper()
	runner.Start()
	done := make(chan struct{})
	go func() {
		_ = runner.stopper.Client().Stopped().Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("runner did not stop within timeout")
	}
}

func TestRunnerUpToDate(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 0})

	runner := newTestRunner(db, &noopRolloutChecker{}, 0)
	requireStoppedWithin(t, runner, testTimeout)
}

func TestRunnerDetectsRollback(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 5})
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(2)).Return(pgconn.CommandTag{}, nil)

	runner := newTestRunner(db, &noopRolloutChecker{}, 2)
	requireStoppedWithin(t, runner, testTimeout)
}

func TestRunnerRunsOnlyNewMigrations(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 1})

	ran := []int{}
	MustRegister(BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "m0",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 0); return nil },
	})
	MustRegister(BackgroundMigration{
		StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "m1",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 1); return nil },
	})
	MustRegister(BackgroundMigration{
		StartingSeqNum: 2, VersionAfterSeqNum: 3, Description: "m2",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 2); return nil },
	})

	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(2)).Return(pgconn.CommandTag{}, nil)
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(3)).Return(pgconn.CommandTag{}, nil)

	runner := newTestRunner(db, &noopRolloutChecker{}, 3)
	requireStoppedWithin(t, runner, testTimeout)

	assert.Equal(t, []int{1, 2}, ran)
}

func TestRunnerStopsOnMigrationError(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 0})

	migErr := errors.New("migration failed")
	MustRegister(BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "fails",
		Run: func(_ context.Context, _ postgres.DB) error { return migErr },
	})
	MustRegister(BackgroundMigration{
		StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "should not run",
		Run: func(_ context.Context, _ postgres.DB) error { t.Fatal("should not have run"); return nil },
	})

	runner := newTestRunner(db, &noopRolloutChecker{}, 2)
	requireStoppedWithin(t, runner, testTimeout)
}

func TestRunnerMigrationRespectsContext(t *testing.T) {
	resetRegistry()
	MustRegister(BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "ctx-aware",
		Run: func(ctx context.Context, _ postgres.DB) error { return ctx.Err() },
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m, ok := Get(0)
	require.True(t, ok)
	err := m.Run(ctx, nil)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRunnerStopCancelsRolloutCheck(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	runner := newTestRunner(db, &blockingRolloutChecker{}, 0)
	runner.Start()

	// Stop should cancel the blocking rollout check and the runner should exit.
	runner.Stop()

	done := make(chan struct{})
	go func() {
		_ = runner.stopper.Client().Stopped().Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("runner did not stop after Stop() within timeout")
	}
}

func TestRunnerLockBlocksUntilStopped(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	runner := newTestRunner(db, &noopRolloutChecker{}, 0)
	runner.tryAcquireLockFunc = neverAcquireLockFunc()
	runner.Start()

	runner.Stop()

	done := make(chan struct{})
	go func() {
		_ = runner.stopper.Client().Stopped().Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("runner did not stop after Stop() within timeout")
	}
}

func TestRunnerLockRetriesOnConnectionDrop(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 0})

	callCount := 0
	runner := newTestRunner(db, &noopRolloutChecker{}, 0)
	runner.tryAcquireLockFunc = func(_ context.Context, _ postgres.DB, _ int64) (bool, func(), error) {
		callCount++
		if callCount < 3 {
			return false, nil, errors.New("connection dropped")
		}
		return true, func() {}, nil
	}

	requireStoppedWithin(t, runner, testTimeout)
	assert.Equal(t, 3, callCount)
}

func TestRunnerLockReleasedAfterMigrations(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 0})

	released := false
	runner := newTestRunner(db, &noopRolloutChecker{}, 0)
	runner.tryAcquireLockFunc = func(_ context.Context, _ postgres.DB, _ int64) (bool, func(), error) {
		return true, func() { released = true }, nil
	}

	requireStoppedWithin(t, runner, testTimeout)
	assert.True(t, released, "advisory lock should be released after runner completes")
}

func TestRunnerStopCancelsRunningMigration(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 0})

	migrationStarted := make(chan struct{})
	MustRegister(BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "long-running",
		Run: func(ctx context.Context, _ postgres.DB) error {
			close(migrationStarted)
			<-ctx.Done()
			return ctx.Err()
		},
	})

	runner := newTestRunner(db, &noopRolloutChecker{}, 1)
	runner.Start()

	select {
	case <-migrationStarted:
	case <-time.After(testTimeout):
		t.Fatal("migration did not start within timeout")
	}

	runner.Stop()

	done := make(chan struct{})
	go func() {
		_ = runner.stopper.Client().Stopped().Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("runner did not stop after cancelling migration within timeout")
	}
}
