//go:build sql_integration

package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/clusterinit/store"
	pgStore "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestClusterInitStore(t *testing.T) {
	suite.Run(t, new(clusterInitStoreTestSuite))
}

type clusterInitStoreTestSuite struct {
	suite.Suite
	store store.Store
	ctx   context.Context
	db    postgres.DB
}

func (s *clusterInitStoreTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *clusterInitStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	s.store = store.NewStore(pgStore.New(s.db))
}

func (s *clusterInitStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *clusterInitStoreTestSuite) TestIDCollisionOnAdd() {
	meta := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "test name",
	}
	idCollision := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "id collision",
	}

	err := s.store.Add(s.ctx, meta)
	s.NoError(err)

	err = s.store.Add(s.ctx, idCollision)
	s.Error(err)
	s.True(errors.Is(err, store.ErrInitBundleIDCollision))
}

func (s *clusterInitStoreTestSuite) TestNameCollisionOnAdd() {
	meta := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "test_name",
	}
	err := s.store.Add(s.ctx, meta)
	s.NoError(err)

	meta2 := &storage.InitBundleMeta{
		Id:   "9876543210",
		Name: "test_name",
	}

	err = s.store.Add(s.ctx, meta2)
	s.Error(err)
}

func (s *clusterInitStoreTestSuite) TestRevokeToken() {
	meta := &storage.InitBundleMeta{
		Id:        "012345",
		Name:      "available",
		IsRevoked: false,
	}
	toRevokeMeta := &storage.InitBundleMeta{
		Id:        "0123456789",
		Name:      "revoked",
		IsRevoked: false,
	}
	toReuseMetaName := &storage.InitBundleMeta{
		Id:        "0123456",
		Name:      "revoked",
		IsRevoked: false,
	}

	for _, m := range []*storage.InitBundleMeta{toRevokeMeta, meta} {
		err := s.store.Add(s.ctx, m)
		s.Require().NoError(err)
	}

	storedMeta, err := s.store.Get(s.ctx, toRevokeMeta.GetId())
	s.Require().NoError(err)
	s.False(storedMeta.GetIsRevoked())

	err = s.store.Revoke(s.ctx, toRevokeMeta.GetId())
	s.Require().NoError(err)

	// test GetAll ignores revoked bundles
	all, err := s.store.GetAll(s.ctx)
	s.Require().NoError(err)
	s.Len(all, 1)
	s.Equal("available", all[0].GetName())

	// test name can be reused after revoking an init-bundle
	err = s.store.Add(s.ctx, toReuseMetaName)
	s.Require().NoError(err)
	reused, err := s.store.Get(s.ctx, toReuseMetaName.GetId())
	s.Require().NoError(err)
	s.Equal(toReuseMetaName.GetName(), reused.GetName())
	s.Equal(toRevokeMeta.GetName(), reused.GetName())
}
