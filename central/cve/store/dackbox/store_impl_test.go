package dackbox

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
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

	store store.Store
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
	suite.store = New(suite.dacky, concurrency.NewKeyFence())
}

func (suite *CVEStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *CVEStoreTestSuite) TestCVES() {
	ctx := sac.WithAllAccess(context.Background())

	cves := []*storage.CVE{
		{
			Id:   "CVE-2019-02-14",
			Cvss: 1.3,
		},
		{
			Id:   "CVE-2019-03-14",
			Cvss: 5.0,
		},
	}

	// Test Add
	for _, d := range cves {
		suite.NoError(suite.store.Upsert(ctx, d))
	}

	for _, d := range cves {
		got, exists, err := suite.store.Get(ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Update
	for _, d := range cves {
		d.Cvss += 1.0
	}

	suite.NoError(suite.store.Upsert(ctx, cves...))

	for _, d := range cves {
		got, exists, err := suite.store.Get(ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Count
	count, err := suite.store.Count(ctx)
	suite.NoError(err)
	suite.Equal(len(cves), count)
}
