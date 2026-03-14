//go:build sql_integration

package lock

import (
	"context"
	"sync"
	"testing"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AdvisoryLockSuite struct {
	suite.Suite
	pool   postgres.DB
	source string
	ctx    context.Context
}

func TestAdvisoryLockSuite(t *testing.T) {
	suite.Run(t, new(AdvisoryLockSuite))
}

func (s *AdvisoryLockSuite) SetupTest() {
	s.ctx = context.Background()

	// Use a single database for all pools in a test so advisory locks are shared.
	s.source = pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(s.source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.Require().NoError(err)
	s.pool = pool
}

func (s *AdvisoryLockSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *AdvisoryLockSuite) TestTryAcquireAndRelease() {
	acquired, release, err := TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)
	s.Require().NotNil(release)

	release()

	acquired2, release2, err := TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired2)
	s.Require().NotNil(release2)
	release2()
}

func (s *AdvisoryLockSuite) TestMutualExclusion() {
	acquired, release, err := TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)
	s.Require().NotNil(release)
	defer release()

	// TryAcquire  should fail because lock already held by other connection.
	acquired2, release2, err := TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().False(acquired2)
	s.Require().Nil(release2)
}

func (s *AdvisoryLockSuite) TestReleaseAllowsReacquire() {
	acquired, release, err := TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)
	release()

	acquired2, release2, err := TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired2)
	s.Require().NotNil(release2)
	release2()
}

func (s *AdvisoryLockSuite) TestConcurrentTryAcquire() {
	// Multiple goroutines compete; exactly one should succeed.
	const numGoroutines = 5
	results := make([]bool, numGoroutines)
	releases := make([]func(), numGoroutines)
	errs := make([]error, numGoroutines)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx], releases[idx], errs[idx] = TryAcquireMigrationLock(s.ctx, s.pool)
		}(i)
	}
	wg.Wait()

	acquiredCount := 0
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(s.T(), errs[i])
		if results[i] {
			acquiredCount++
			require.NotNil(s.T(), releases[i])
			releases[i]()
		}
	}
	assert.Equal(s.T(), 1, acquiredCount, "exactly one goroutine should acquire the lock")
}

func (s *AdvisoryLockSuite) TestDoubleReleaseIsIdempotent() {
	acquired, release, err := TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)

	// Double release should not panic.
	release()
	release()
}
