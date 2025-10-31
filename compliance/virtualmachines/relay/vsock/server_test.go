package vsock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

func TestVsockServer(t *testing.T) {
	suite.Run(t, new(serverTestSuite))
}

type serverTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *serverTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *serverTestSuite) TestSemaphore() {
	vsockServer := &serverImpl{
		semaphore:        semaphore.NewWeighted(1),
		semaphoreTimeout: 5 * time.Millisecond,
	}

	// First should succeed
	err := vsockServer.AcquireSemaphore(s.ctx)
	s.Require().NoError(err)

	// Second should time out
	err = vsockServer.AcquireSemaphore(s.ctx)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to acquire semaphore")

	// After releasing once, a new acquire should succeed
	vsockServer.ReleaseSemaphore()
	err = vsockServer.AcquireSemaphore(s.ctx)
	s.Require().NoError(err)
}
