//go:build sql_integration

package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

type WorkerMigrationsSuite struct {
	suite.Suite
	pool postgres.DB
	ctx  context.Context
}

func TestWorkerMigrations(t *testing.T) {
	suite.Run(t, new(WorkerMigrationsSuite))
}

func (s *WorkerMigrationsSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(s.T().Context())
	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	s.pool, err = postgres.New(s.ctx, config)
	s.Require().NoError(err)
	s.T().Cleanup(s.pool.Close)
	gormDB := pgtest.OpenGormDB(s.T(), source)
	s.T().Cleanup(func() { pgtest.CloseGormDB(s.T(), gormDB) })
	schema.ApplySchemaForTable(s.ctx, gormDB, schema.VersionsSchema.Table)
}

func (s *WorkerMigrationsSuite) setVersion(seqNum int) {
	s.T().Helper()
	s.Require().NoError(vStore.NewPostgres(s.pool).UpdateVersion(&storage.Version{SeqNum: int32(seqNum)}))
}

func (s *WorkerMigrationsSuite) holdLock(id int64) func() {
	s.T().Helper()
	acquired, release, err := dblock.TryAcquireAdvisoryLock(s.ctx, s.pool, id)
	s.Require().NoError(err)
	s.Require().True(acquired)
	s.T().Cleanup(release)
	return release
}

type countingPool struct {
	postgres.DB
	attempts atomic.Int32
}

func (p *countingPool) Acquire(ctx context.Context) (*postgres.Conn, error) {
	p.attempts.Add(1)
	return p.DB.Acquire(ctx)
}

func (s *WorkerMigrationsSuite) TestWaitBeyondOriginalRetryLimit() {
	s.setVersion(migrations.CurrentDBVersionSeqNum())
	release := s.holdLock(dblock.MigrationLockID)
	pool := &countingPool{DB: s.pool}
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- waitForMigrations(ctx, pool, time.Millisecond) }()

	// A current version is insufficient while the migrator still holds its
	// lock. Waiting must continue past the previous limit of 30 attempts.
	s.Require().Eventually(func() bool { return pool.attempts.Load() > 30 }, 3*time.Second, time.Millisecond)
	select {
	case err := <-result:
		s.FailNow("Worker stopped waiting while startup migrations held the lock", "%v", err)
	default:
	}
	release()
	select {
	case err := <-result:
		s.Require().NoError(err)
	case <-ctx.Done():
		s.FailNow("Worker did not start after startup migrations finished")
	}
}

func (s *WorkerMigrationsSuite) TestMigrationVersionNotReady() {
	for name, seqNum := range map[string]int{
		"missing version": -1,
		"older version":   migrations.CurrentDBVersionSeqNum() - 1,
	} {
		s.Run(name, func() {
			_, err := s.pool.Exec(s.ctx, "DELETE FROM versions")
			s.Require().NoError(err)
			if seqNum >= 0 {
				s.setVersion(seqNum)
			}
			s.Require().Error(checkMigrations(s.ctx, s.pool))
			// A failed version check must release the lock so Central can migrate.
			release := s.holdLock(dblock.MigrationLockID)
			release()
		})
	}
}

func (s *WorkerMigrationsSuite) TestWaitsForVersionPublication() {
	s.setVersion(migrations.CurrentDBVersionSeqNum() - 1)
	pool := &countingPool{DB: s.pool}
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- waitForMigrations(ctx, pool, time.Millisecond) }()
	s.Require().Eventually(func() bool { return pool.attempts.Load() > 1 }, 3*time.Second, time.Millisecond)
	s.setVersion(migrations.CurrentDBVersionSeqNum())
	select {
	case err := <-result:
		s.Require().NoError(err)
	case <-ctx.Done():
		s.FailNow("Worker did not start after version publication")
	}
}

func (s *WorkerMigrationsSuite) TestCompatibleVersion() {
	for name, seqNum := range map[string]int{
		"current": migrations.CurrentDBVersionSeqNum(),
		"newer":   migrations.CurrentDBVersionSeqNum() + 1,
	} {
		s.Run(name, func() {
			s.setVersion(seqNum)
			s.Require().NoError(checkMigrations(s.ctx, s.pool))
		})
	}
}

func (s *WorkerMigrationsSuite) TestShutdownCancelsMigrationWait() {
	s.holdLock(dblock.MigrationLockID)
	ctx, cancel := context.WithCancel(s.ctx)
	timer := time.AfterFunc(20*time.Millisecond, cancel)
	defer timer.Stop()
	defer cancel()
	// Cancellation must interrupt the retry interval, rather than waiting an hour.
	s.Require().ErrorIs(waitForMigrations(ctx, s.pool, time.Hour), context.Canceled)
}

func (s *WorkerMigrationsSuite) TestBackgroundMigrationDoesNotBlockStartup() {
	s.setVersion(migrations.CurrentDBVersionSeqNum())
	s.holdLock(dblock.BackgroundMigrationLockID)
	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()
	s.Require().NoError(waitForMigrations(ctx, s.pool, time.Millisecond))
}
