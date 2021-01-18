package backend

import (
	"context"
	"testing"

	rocksdbStore "github.com/stackrox/rox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
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
	rocksDB *rocksdb.RocksDB
}

func (s *clusterInitBackendTestSuite) SetupTest() {
	s.rocksDB = rocksdbtest.RocksDBForT(s.T())
	store, err := rocksdbStore.NewStore(s.rocksDB)
	s.Require().NoError(err)
	s.backend = newBackend(store)
	s.ctx = context.Background()
}

func (s *clusterInitBackendTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksDB)
}
