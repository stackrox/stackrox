package tests

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type storeCreator func(t *testing.T) (store.Store, func(t *testing.T))

func createRocksDBStore(t *testing.T) (store.Store, func(t *testing.T)) {
	testRocksDB := rocksdbtest.RocksDBForT(t)
	rocksStore, err := rocksdb.NewStore(testRocksDB)
	require.NoError(t, err)

	tearDown := func(t *testing.T) {
		rocksdbtest.TearDownRocksDB(testRocksDB)
	}
	return rocksStore, tearDown
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
}

func (s *clusterInitStoreTestSuite) SetupTest() {
	s.store, s.teardownStore = s.storeCreator(s.T())
}

func (s *clusterInitStoreTestSuite) TearDownTest() {
	s.teardownStore(s.T())
}

func (s *clusterInitStoreTestSuite) TestIDCollisionOnAdd() {
	meta := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "test name",
	}
	err := s.store.Add(meta)
	s.NoError(err)

	err = s.store.Add(meta)
	s.Error(err)
	s.True(errors.Is(err, store.ErrInitBundleIDCollision))
}

func (s *clusterInitStoreTestSuite) TestNameCollisionOnAdd() {
	meta := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "test_name",
	}
	err := s.store.Add(meta)
	s.NoError(err)

	meta2 := &storage.InitBundleMeta{
		Id:   "9876543210",
		Name: "test_name",
	}

	err = s.store.Add(meta2)
	s.Error(err)
}
