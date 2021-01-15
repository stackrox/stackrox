package store

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestClusterInitStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterInitStoreTestSuite))
}

type clusterInitStoreTestSuite struct {
	suite.Suite
	store Store
	ctx   context.Context
}

func (s *clusterInitStoreTestSuite) SetupTest() {
	s.store = NewInMemory()
	s.ctx = context.Background()
}

func (s *clusterInitStoreTestSuite) TestIDCollisionOnAdd() {
	meta := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "test name",
	}
	err := s.store.Add(s.ctx, meta)
	s.NoError(err)

	err = s.store.Add(s.ctx, meta)
	s.Error(err)
	s.True(errors.Is(err, ErrInitBundleIDCollision))
}
