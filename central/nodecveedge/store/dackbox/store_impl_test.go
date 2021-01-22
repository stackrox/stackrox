package dackbox

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/nodecveedge/store"
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
		suite.FailNow("Failed to create counter", err.Error())
	}
	suite.store = New(suite.dacky)
}

func (suite *EdgeStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *EdgeStoreTestSuite) TestNodes() {
	ts1 := types.TimestampNow()
	edges := []*storage.NodeCVEEdge{
		{
			Id:                  "Node1CVE1",
			FirstNodeOccurrence: ts1,
		},
		{
			Id:                  "Node2CVE2",
			FirstNodeOccurrence: ts1,
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

	// Test Update
	ts2 := &types.Timestamp{Seconds: ts1.Seconds + 5, Nanos: ts1.Nanos}
	for _, d := range edges {
		d.FirstNodeOccurrence = ts2
	}

	suite.NoError(suite.store.Upsert(edges...))

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
