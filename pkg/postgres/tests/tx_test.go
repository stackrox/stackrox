package tests

import (
	"context"
	"testing"
	"time"

	pkgmocks "github.com/stackrox/rox/pkg/mocks/github.com/jackc/pgx/v5/mocks"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestTx(t *testing.T) {
	suite.Run(t, new(postgresTxTestSuite))
}

type postgresTxTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	mockTx   *pkgmocks.MockTx
	tx       *postgres.Tx
}

func (s *postgresTxTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *postgresTxTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *postgresTxTestSuite) SetupTest() {
	s.mockTx = pkgmocks.NewMockTx(s.mockCtrl)
	s.tx = &postgres.Tx{}
	s.tx.Tx = s.mockTx
}

// TestCommitWithExpiredContext verifies that Commit succeeds even when called with an expired context.
func (s *postgresTxTestSuite) TestCommitWithExpiredContext() {
	// Create an already-expired context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	s.Error(ctx.Err(), "context should be expired")

	s.mockTx.EXPECT().Commit(gomock.Any()).Times(1).Return(nil)

	err := s.tx.Commit(ctx)
	s.NoError(err)
}

// TestRollbackWithExpiredContext verifies that Rollback succeeds even when called with an expired context.
func (s *postgresTxTestSuite) TestRollbackWithExpiredContext() {
	// Create an already-expired context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	s.Error(ctx.Err(), "context should be expired")

	s.mockTx.EXPECT().Rollback(gomock.Any()).Times(1).Return(nil)

	err := s.tx.Rollback(ctx)
	s.NoError(err)
}

// TestCommitWithValidContext verifies normal Commit behavior works
func (s *postgresTxTestSuite) TestCommitWithValidContext() {
	ctx := context.Background()

	s.mockTx.EXPECT().Commit(gomock.Any()).Times(1).Return(nil)

	err := s.tx.Commit(ctx)
	s.NoError(err)
}

// TestRollbackWithValidContext verifies normal Rollback behavior works
func (s *postgresTxTestSuite) TestRollbackWithValidContext() {
	ctx := context.Background()

	s.mockTx.EXPECT().Rollback(gomock.Any()).Times(1).Return(nil)

	err := s.tx.Rollback(ctx)
	s.NoError(err)
}
