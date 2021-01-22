package dackbox

import (
	"testing"

	"github.com/stackrox/rox/central/nodecomponentedge/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestEdgeStore(t *testing.T) {
	suite.Run(t, new(EdgeStoreTestSuite))
}

type EdgeStoreTestSuite struct {
	suite.Suite

	db    *rocksdb.RocksDB
	dacky *dackbox.DackBox

	store store.Store
}

func (suite *EdgeStoreTestSuite) SetupSuite() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())
	var err error
	suite.dacky, err = dackbox.NewRocksDBDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNow("failed to create DackBox", err.Error())
	}
	suite.store = New(suite.dacky)
}

func (suite *EdgeStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *EdgeStoreTestSuite) TestNodes() {
	edges := []*storage.NodeComponentEdge{
		{
			Id: "CVE-2019-02-14",
		},
		{
			Id: "CVE-2019-03-14",
		},
	}

	// Test Add
	for _, d := range edges {
		suite.NoError(suite.store.Upsert(d))
	}

	for _, d := range edges {
		got, exists, err := suite.store.Get(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Count
	count, err := suite.store.Count()
	suite.NoError(err)
	suite.Equal(len(edges), count)
}
