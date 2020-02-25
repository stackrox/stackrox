package dackbox

import (
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestCVEStore(t *testing.T) {
	suite.Run(t, new(CVEStoreTestSuite))
}

type CVEStoreTestSuite struct {
	suite.Suite

	db    *badger.DB
	dir   string
	dacky *dackbox.DackBox

	store store.Store
}

func (suite *CVEStoreTestSuite) SetupSuite() {
	var err error
	suite.db, suite.dir, err = badgerhelper.NewTemp("reference")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
	suite.dacky, err = dackbox.NewDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
	suite.store, err = New(suite.dacky, concurrency.NewKeyFence())
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
}

func (suite *CVEStoreTestSuite) TearDownSuite() {
	testutils.TearDownBadger(suite.db, suite.dir)
}

func (suite *CVEStoreTestSuite) TestCVES() {
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
		suite.NoError(suite.store.Upsert(d))
	}

	for _, d := range cves {
		got, exists, err := suite.store.Get(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Update
	for _, d := range cves {
		d.Cvss += 1.0
	}

	suite.NoError(suite.store.Upsert(cves...))

	for _, d := range cves {
		got, exists, err := suite.store.Get(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Count
	count, err := suite.store.Count()
	suite.NoError(err)
	suite.Equal(len(cves), count)
}

func (suite *CVEStoreTestSuite) TestClusterCVES() {
	cveParts := []converter.ClusterCVEParts{
		{
			CVE: &storage.CVE{
				Id:   "CVE-2019-02-14",
				Cvss: 1.3,
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
		suite.NoError(suite.store.UpsertClusterCVEs(d))
	}

	for _, d := range cveParts {
		got, exists, err := suite.store.Get(d.CVE.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d.CVE)
	}

	// Test Update
	for _, d := range cveParts {
		d.CVE.Cvss += 1.0
	}

	suite.NoError(suite.store.UpsertClusterCVEs(cveParts...))

	for _, d := range cveParts {
		got, exists, err := suite.store.Get(d.CVE.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d.CVE)
	}

	// Test Count
	count, err := suite.store.Count()
	suite.NoError(err)
	suite.Equal(len(cveParts), count)
}
