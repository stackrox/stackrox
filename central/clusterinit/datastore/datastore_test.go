package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestClusterInitDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterInitDataStoreTestSuite))
}

type clusterInitDataStoreTestSuite struct {
	suite.Suite
	dataStore DataStore
	ctx       context.Context
}

func (s *clusterInitDataStoreTestSuite) SetupTest() {
	s.dataStore = NewInMemory()
	s.ctx = context.Background()
}

func (s *clusterInitDataStoreTestSuite) TestTokenIDCollision() {
	tokenMeta := &storage.BootstrapTokenWithMeta{
		Id:          "0123456789",
		Token:       []byte("very secret test bootstrap token"),
		Description: "test description",
	}
	err := s.dataStore.Add(s.ctx, tokenMeta)
	s.NoError(err)

	err = s.dataStore.Add(s.ctx, tokenMeta)
	s.Error(err)
	s.True(errors.Is(err, ErrTokenIDCollision))
}
