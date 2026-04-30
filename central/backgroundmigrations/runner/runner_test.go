//go:build sql_integration

package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const testTimeout = 1 * time.Second

type doneRolloutChecker struct{}

func (n *doneRolloutChecker) IsRolloutDone(_ context.Context) (bool, error) { return true, nil }

type notDoneRolloutChecker struct{}

func (c *notDoneRolloutChecker) IsRolloutDone(_ context.Context) (bool, error) { return false, nil }

// countingRolloutChecker returns not-done for the first N calls, then done.
type countingRolloutChecker struct {
	notDoneCount int
	callCount    int
}

func (c *countingRolloutChecker) IsRolloutDone(_ context.Context) (bool, error) {
	c.callCount++
	if c.callCount <= c.notDoneCount {
		return false, errors.New("transient rollout check error")
	}
	return true, nil
}

var _ RolloutChecker = &doneRolloutChecker{}
var _ RolloutChecker = &k8sRolloutChecker{}
var _ RolloutChecker = &notDoneRolloutChecker{}
var _ RolloutChecker = &countingRolloutChecker{}

type RunnerTestSuite struct {
	suite.Suite
	db  postgres.DB
	ctx context.Context
}

func TestRunnerSuite(t *testing.T) {
	suite.Run(t, new(RunnerTestSuite))
}

func (s *RunnerTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Create a test database with a pool that supports multiple connections.
	// The advisory lock holds one connection, so we need at least 2.
	database := pgtest.CreateADatabaseForT(s.T())
	source := conn.GetConnectionStringWithDatabaseName(s.T(), database)
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		pool.Close()
		pgtest.DropDatabase(s.T(), database)
	})
	s.db = pool

	_, err = pool.Exec(s.ctx,
		"CREATE TABLE IF NOT EXISTS "+schema.BackgroundMigrationVersionsTableName+
			" (seqnum integer PRIMARY KEY NOT NULL, override_tag text DEFAULT '')")
	s.Require().NoError(err)
}

func (s *RunnerTestSuite) SetupTest() {
	migrations.ResetRegistryForTesting(s.T())

	_, err := s.db.Exec(s.ctx,
		"DELETE FROM "+schema.BackgroundMigrationVersionsTableName)
	s.Require().NoError(err)
	_, err = s.db.Exec(s.ctx,
		"INSERT INTO "+schema.BackgroundMigrationVersionsTableName+" (seqnum, override_tag) VALUES (0, '')")
	s.Require().NoError(err)
}

func (s *RunnerTestSuite) newRunner(rolloutChecker RolloutChecker, targetSeqNum int) *Runner {
	r := NewRunner(s.db, rolloutChecker)
	r.targetSeqNum = targetSeqNum
	r.retryInterval = 10 * time.Millisecond
	return r
}

// requireStoppedWithin starts the runner and fails the test if it doesn't stop within the timeout.
func requireStoppedWithin(t *testing.T, runner *Runner, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		_ = runner.stopper.Client().Stopped().Wait()
		close(done)
	}()
	runner.Start()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("runner did not stop within timeout")
	}
}

func (s *RunnerTestSuite) TestUpToDate() {
	runner := s.newRunner(&doneRolloutChecker{}, 0)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, _, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(0, seqNum)
}

func (s *RunnerTestSuite) TestDetectsRollback() {
	// Set DB to seq 5, but current is 2.
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 5")
	s.Require().NoError(err)

	runner := s.newRunner(&doneRolloutChecker{}, 2)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, _, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(2, seqNum)
}

func (s *RunnerTestSuite) TestRunsOnlyNewMigrations() {
	// DB at seq 1, current target is 3 — should only run migrations 1 and 2.
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 1")
	s.Require().NoError(err)

	ran := []int{}
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "m0",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 0); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "m1",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 1); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 2, VersionAfterSeqNum: 3, Description: "m2",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 2); return nil },
	})

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	s.Equal([]int{1, 2}, ran)

	seqNum, _, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
}

func (s *RunnerTestSuite) TestRetriesOnMigrationError() {
	callCount := 0
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "fails-then-succeeds",
		Run: func(_ context.Context, _ postgres.DB) error {
			callCount++
			if callCount == 1 {
				return errors.New("transient failure")
			}
			return nil
		},
	})

	runner := s.newRunner(&doneRolloutChecker{}, 1)
	requireStoppedWithin(s.T(), runner, testTimeout)

	s.Equal(2, callCount)
}

func (s *RunnerTestSuite) TestRetryStopsOnShutdown() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "always-fails",
		Run: func(_ context.Context, _ postgres.DB) error { return errors.New("permanent failure") },
	})

	runner := s.newRunner(&doneRolloutChecker{}, 1)
	runner.Start()

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
		s.T().Fatal("runner did not stop after Stop() within timeout")
	}
}

func (s *RunnerTestSuite) TestMigrationRespectsContext() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "ctx-aware",
		Run: func(ctx context.Context, _ postgres.DB) error { return ctx.Err() },
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m, ok := migrations.Get(0)
	require.True(s.T(), ok)
	err := m.Run(ctx, nil)
	assert.ErrorIs(s.T(), err, context.Canceled)
}

func (s *RunnerTestSuite) TestStopDuringRolloutRetry() {
	runner := s.newRunner(&notDoneRolloutChecker{}, 0)
	runner.Start()

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
		s.T().Fatal("runner did not stop after Stop() within timeout")
	}
}

func (s *RunnerTestSuite) TestRetriesOnRolloutCheckError() {
	checker := &countingRolloutChecker{notDoneCount: 2}
	runner := s.newRunner(checker, 0)
	requireStoppedWithin(s.T(), runner, testTimeout)

	s.Equal(3, checker.callCount)
}

func (s *RunnerTestSuite) TestLockReleasedAfterMigrations() {
	runner := s.newRunner(&doneRolloutChecker{}, 0)
	requireStoppedWithin(s.T(), runner, testTimeout)

	// If the lock was released, we should be able to acquire it again.
	runner2 := s.newRunner(&doneRolloutChecker{}, 0)
	requireStoppedWithin(s.T(), runner2, testTimeout)
}

func (s *RunnerTestSuite) TestStopCancelsRunningMigration() {
	migrationStarted := make(chan struct{})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "long-running",
		Run: func(ctx context.Context, _ postgres.DB) error {
			close(migrationStarted)
			<-ctx.Done()
			return ctx.Err()
		},
	})

	runner := s.newRunner(&doneRolloutChecker{}, 1)
	runner.Start()

	select {
	case <-migrationStarted:
	case <-time.After(testTimeout):
		s.T().Fatal("migration did not start within timeout")
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
		s.T().Fatal("runner did not stop after cancelling migration within timeout")
	}
}

func (s *RunnerTestSuite) TestOverrideAppliesWithNewTag() {
	// DB at seq 3, override resets to 0.
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3")
	s.Require().NoError(err)

	ran := []int{}
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "m0",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 0); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "m1",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 1); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 2, VersionAfterSeqNum: 3, Description: "m2",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 2); return nil },
	})

	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-123")

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	s.Equal([]int{0, 1, 2}, ran)

	_, overrideTag, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal("ROX-123", overrideTag)
}

func (s *RunnerTestSuite) TestOverrideSkipsWhenTagMatches() {
	// DB already has the same tag — override was already applied.
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3, override_tag = 'ROX-123'")
	s.Require().NoError(err)

	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-123")

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, _, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
}

func (s *RunnerTestSuite) TestOverrideRerunsWithNewTag() {
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3, override_tag = 'ROX-123'")
	s.Require().NoError(err)

	ran := []int{}
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "m0",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 0); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "m1",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 1); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 2, VersionAfterSeqNum: 3, Description: "m2",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 2); return nil },
	})

	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-456")

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	s.Equal([]int{0, 1, 2}, ran)
}

func (s *RunnerTestSuite) TestOverrideIgnoredWithoutTag() {
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3")
	s.Require().NoError(err)

	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, _, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
}

func (s *RunnerTestSuite) TestOverrideIgnoredWhenSeqNumExceedsCurrent() {
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3")
	s.Require().NoError(err)

	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "10")
	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-999")

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, _, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
}

func (s *RunnerTestSuite) TestSeedsInitialRowWhenTableEmpty() {
	_, err := s.db.Exec(s.ctx, "DELETE FROM "+schema.BackgroundMigrationVersionsTableName)
	s.Require().NoError(err)

	runner := s.newRunner(&doneRolloutChecker{}, 0)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, overrideTag, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(0, seqNum)
	s.Equal("", overrideTag)
}

func (s *RunnerTestSuite) TestOverrideTagClearedWhenEnvRemoved() {
	// DB has a stale override tag from a previous run, but env vars are not set.
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3, override_tag = 'ROX-123'")
	s.Require().NoError(err)

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, overrideTag, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
	s.Equal("", overrideTag)
}

func (s *RunnerTestSuite) TestOverrideNotReappliedOnRestart() {
	// Simulates the restart loop bug: env vars are still set, tag matches DB.
	// First run applies the override and re-runs migrations (expected).
	// Second run should NOT re-run migrations because the tag already matches.
	ran := []int{}
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "m0",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 0); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "m1",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 1); return nil },
	})
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum: 2, VersionAfterSeqNum: 3, Description: "m2",
		Run: func(_ context.Context, _ postgres.DB) error { ran = append(ran, 2); return nil },
	})

	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3")
	s.Require().NoError(err)

	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", "0")
	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-123")

	// First run: override applies, migrations re-run.
	runner1 := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner1, testTimeout)
	s.Equal([]int{0, 1, 2}, ran)

	seqNum, overrideTag, err := runner1.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
	s.Equal("ROX-123", overrideTag, "override tag must be preserved after first run")

	// Second run (simulating restart): env vars still set, tag matches DB.
	// Migrations must NOT re-run.
	ran = []int{}
	runner2 := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner2, testTimeout)
	s.Empty(ran, "migrations must not re-run when override tag already matches")

	seqNum, overrideTag, err = runner2.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
	s.Equal("ROX-123", overrideTag, "override tag must remain after second run")
}

func (s *RunnerTestSuite) TestOverrideIgnoredWithoutSeqNum() {
	_, err := s.db.Exec(s.ctx,
		"UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = 3")
	s.Require().NoError(err)

	s.T().Setenv("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG", "ROX-123")

	runner := s.newRunner(&doneRolloutChecker{}, 3)
	requireStoppedWithin(s.T(), runner, testTimeout)

	seqNum, _, err := runner.readState(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, seqNum)
}
