package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/stackrox/central/clusterinit/store"
	"github.com/stackrox/stackrox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type storeCreator func(t *testing.T) (store.Store, func(t *testing.T))

func createRocksDBStore(t *testing.T) (store.Store, func(t *testing.T)) {
	testRocksDB := rocksdbtest.RocksDBForT(t)
	rocksStore, err := rocksdb.New(testRocksDB)
	require.NoError(t, err)
	tearDown := func(t *testing.T) {
		rocksdbtest.TearDownRocksDB(testRocksDB)
	}
	return store.NewStore(rocksStore), tearDown
}

func TestClusterInitStore(t *testing.T) {
	t.Parallel()

	stores := map[string]storeCreator{
		"rocksdb": createRocksDBStore,
	}

	for name, storeCreator := range stores {
		t.Run(name, func(t *testing.T) {
			suite.Run(t, &clusterInitStoreTestSuite{storeCreator: storeCreator})
		})
	}
}

type clusterInitStoreTestSuite struct {
	storeCreator storeCreator

	suite.Suite
	store         store.Store
	teardownStore func(t *testing.T)
	ctx           context.Context
	cancel        context.CancelFunc
}

func (s *clusterInitStoreTestSuite) SetupTest() {
	s.store, s.teardownStore = s.storeCreator(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *clusterInitStoreTestSuite) TearDownTest() {
	s.teardownStore(s.T())
	s.cancel()
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
