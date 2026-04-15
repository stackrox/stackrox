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

// fakeRow implements pgx.Row for returning state from QueryRow.
type fakeRow struct {
	seqNum      int32
	overrideTag string
	err         error
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
	if len(dest) > 1 {
		if p, ok := dest[1].(*string); ok {
			*p = r.overrideTag
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

func TestRunnerRetriesOnMigrationError(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	// First attempt: read state, migration fails.
	// Second attempt: read state, migration succeeds.
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 0}).Times(2)

	callCount := 0
	MustRegister(BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "fails-then-succeeds",
		Run: func(_ context.Context, _ postgres.DB) error {
			callCount++
			if callCount == 1 {
				return errors.New("transient failure")
			}
			return nil
		},
	})

	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(1)).Return(pgconn.CommandTag{}, nil)

	runner := newTestRunner(db, &noopRolloutChecker{}, 1)
	requireStoppedWithin(t, runner, testTimeout)

	assert.Equal(t, 2, callCount)
}

func TestRunnerRetryStopsOnShutdown(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 0}).AnyTimes()

	MustRegister(BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "always-fails",
		Run: func(_ context.Context, _ postgres.DB) error { return errors.New("permanent failure") },
	})

	runner := newTestRunner(db, &noopRolloutChecker{}, 1)
	runner.Start()

	// Let it fail at least once, then stop.
	time.Sleep(50 * time.Millisecond)
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

func TestRunnerOverrideAppliesWithNewTag(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	// DB is at seq 3, override wants to restart from 0 with tag "ROX-123".
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 3, overrideTag: ""})
	// Expect writeState to persist the override.
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(0), "ROX-123").Return(pgconn.CommandTag{}, nil)

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

	// Each migration writes its seq num.
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(1)).Return(pgconn.CommandTag{}, nil)
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(2)).Return(pgconn.CommandTag{}, nil)
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(3)).Return(pgconn.CommandTag{}, nil)

	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-123")

	runner := newTestRunner(db, &noopRolloutChecker{}, 3)
	requireStoppedWithin(t, runner, testTimeout)

	assert.Equal(t, []int{0, 1, 2}, ran)
}

func TestRunnerOverrideSkipsWhenTagMatches(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	// DB already has the same tag — override was already applied by another replica.
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 3, overrideTag: "ROX-123"})
	// No writeState expected — override is skipped.

	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-123")

	runner := newTestRunner(db, &noopRolloutChecker{}, 3)
	requireStoppedWithin(t, runner, testTimeout)
}

func TestRunnerOverrideRerunsWithNewTag(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	// DB has old tag "ROX-123", new tag "ROX-456" triggers a rerun.
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 3, overrideTag: "ROX-123"})
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(0), "ROX-456").Return(pgconn.CommandTag{}, nil)

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

	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(1)).Return(pgconn.CommandTag{}, nil)
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(2)).Return(pgconn.CommandTag{}, nil)
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), int32(3)).Return(pgconn.CommandTag{}, nil)

	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-456")

	runner := newTestRunner(db, &noopRolloutChecker{}, 3)
	requireStoppedWithin(t, runner, testTimeout)

	assert.Equal(t, []int{0, 1, 2}, ran)
}

func TestRunnerOverrideIgnoredWithoutTag(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	// Override seq num set but no tag — override should be ignored.
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 3})

	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	// ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG not set.

	runner := newTestRunner(db, &noopRolloutChecker{}, 3)
	requireStoppedWithin(t, runner, testTimeout)
}

func TestRunnerOverrideIgnoredWhenSeqNumExceedsCurrent(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	// Override seq num (10) exceeds current (3) — override should be ignored.
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 3})
	// No writeState expected.

	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "10")
	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-999")

	runner := newTestRunner(db, &noopRolloutChecker{}, 3)
	requireStoppedWithin(t, runner, testTimeout)
}

func TestRunnerOverrideIgnoredWithoutSeqNum(t *testing.T) {
	resetRegistry()
	ctrl := gomock.NewController(t)
	db := pgMocks.NewMockDB(ctrl)

	// Tag set but no seq num — override should be ignored.
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(&fakeRow{seqNum: 3})

	t.Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-123")
	// ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM not set (defaults to -1).

	runner := newTestRunner(db, &noopRolloutChecker{}, 3)
	requireStoppedWithin(t, runner, testTimeout)
}
