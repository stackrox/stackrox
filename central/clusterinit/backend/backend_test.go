package backend

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestClusterInitBackend(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterInitBackendTestSuite))
}

type clusterInitBackendTestSuite struct {
	suite.Suite
	backend Backend
	ctx     context.Context
}

func (s *clusterInitBackendTestSuite) SetupTest() {
	dataStore := datastore.NewInMemory()
	s.backend = newBackend(dataStore)
	s.ctx = context.Background()
}

func (s *clusterInitBackendTestSuite) TestTokenLifecycle() {
	description := "test description"

	// Issue new token.
	tokenMeta, err := s.backend.Issue(s.ctx, description)
	s.NoError(err)

	tokenID := tokenMeta.GetId()
	token := tokenMeta.GetToken()

	s.Greater(len(tokenID), 0)
	s.Greater(len(token), 0)
	s.Equal(tokenMeta.GetDescription(), description)

	// Test Get.
	tokenMetaRetrieved, err := s.backend.Get(s.ctx, tokenID)
	s.NoError(err)
	s.Equal(tokenMeta, tokenMetaRetrieved)

	// Test GetAll.
	allTokenMetas, err := s.backend.GetAll(s.ctx)
	s.NoError(err)
	s.Equal(allTokenMetas, []*storage.BootstrapTokenWithMeta{tokenMeta})

	// Test Revoke.
	err = s.backend.Revoke(s.ctx, tokenID)
	s.NoError(err)

	// Test Revoke again, should fail.
	err = s.backend.Revoke(s.ctx, tokenID)
	s.Error(err)
	s.True(errors.Is(err, datastore.ErrTokenNotFound))

	// Test Get, should fail.
	_, err = s.backend.Get(s.ctx, tokenID)
	s.Error(err)
	s.True(errors.Is(err, datastore.ErrTokenNotFound))

	// Test GetAll, should be empty.
	allTokenMetas, err = s.backend.GetAll(s.ctx)
	s.NoError(err)
	s.Empty(allTokenMetas)
}

func (s *clusterInitBackendTestSuite) TestTokenCanBeDeactivated() {
	description := "test description"

	// Issue new token.
	tokenMeta, err := s.backend.Issue(s.ctx, description)
	s.NoError(err)

	tokenID := tokenMeta.GetId()
	s.True(tokenMeta.GetActive())

	err = s.backend.SetActive(s.ctx, tokenID, false)
	s.NoError(err)
	tokenMetaRetrieved, err := s.backend.Get(s.ctx, tokenID)
	s.NoError(err)
	s.False(tokenMetaRetrieved.GetActive())

	err = s.backend.SetActive(s.ctx, tokenID, true)
	s.NoError(err)
	tokenMetaRetrieved, err = s.backend.Get(s.ctx, tokenID)
	s.NoError(err)
	s.True(tokenMetaRetrieved.GetActive())
}
