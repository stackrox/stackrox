package dackbox

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/cve/converter"
	cveStore "github.com/stackrox/rox/central/cve/store"
	cveStoreDackBox "github.com/stackrox/rox/central/cve/store/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestCVEStore(t *testing.T) {
	suite.Run(t, new(CVEStoreTestSuite))
}

type CVEStoreTestSuite struct {
	suite.Suite

	db    *rocksdb.RocksDB
	dacky *dackbox.DackBox

	store    store.Store
	cveStore cveStore.Store
}

func (suite *CVEStoreTestSuite) SetupSuite() {
	var err error
	suite.db, err = rocksdb.NewTemp("reference")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
	suite.dacky, err = dackbox.NewRocksDBDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
	suite.store, err = New(suite.dacky, concurrency.NewKeyFence())
	suite.Assert().NoError(err)
	suite.cveStore = cveStoreDackBox.New(suite.dacky, concurrency.NewKeyFence())
}

func (suite *CVEStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *CVEStoreTestSuite) TestClusterCVES() {
	ctx := sac.WithAllAccess(context.Background())

	cveParts := []converter.ClusterCVEParts{
		{
			CVE: &storage.CVE{
				Id:   "CVE-2019-02-14",
				Cvss: 1.3,
			},
			Children: []converter.EdgeParts{
				{
					ClusterID: "cluster",
					Edge: &storage.ClusterCVEEdge{
						Id: edges.EdgeID{ParentID: "test_cluster_id1", ChildID: "CVE-1"}.ToString(),
					},
				},
			},
		},
		{
			CVE: &storage.CVE{
				Id:   "CVE-2019-03-14",
				Cvss: 5.0,
			},
		},
	}

	// Test Add
	for _, d := range cveParts {
		suite.NoError(suite.store.Upsert(ctx, d))
	}

	for _, d := range cveParts {
		got, exists, err := suite.cveStore.Get(ctx, d.CVE.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d.CVE)
	}

	// Test Update
	for _, d := range cveParts {
		d.CVE.Cvss += 1.0
	}

	suite.NoError(suite.store.Upsert(ctx, cveParts...))

	for _, d := range cveParts {
		got, exists, err := suite.cveStore.Get(ctx, d.CVE.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d.CVE)
	}

	// Test Count
	count, err := suite.cveStore.Count(ctx)
	suite.NoError(err)
	suite.Equal(len(cveParts), count)
	edge, exists, err := suite.store.Get(ctx, cveParts[0].Children[0].Edge.Id)
	suite.NoError(err)
	suite.True(exists)

	suite.NoError(suite.store.Delete(ctx, edge.Id))
	_, exists, err = suite.store.Get(ctx, cveParts[0].Children[0].Edge.Id)
	suite.NoError(err)
	suite.False(exists)
}
