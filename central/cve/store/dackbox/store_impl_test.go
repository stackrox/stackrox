package dackbox

import (
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/cve/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestImageStore(t *testing.T) {
	if !features.ManagedDB.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(ImageStoreTestSuite))
}

type ImageStoreTestSuite struct {
	suite.Suite

	db    *badger.DB
	dir   string
	dacky *dackbox.DackBox

	store store.Store
}

func (suite *ImageStoreTestSuite) SetupSuite() {
	var err error
	suite.db, suite.dir, err = badgerhelper.NewTemp("reference", true)
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
	suite.dacky, err = dackbox.NewDackBox(suite.db, []byte("ref_"))
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
	suite.store, err = New(suite.dacky)
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
}

func (suite *ImageStoreTestSuite) TearDownSuite() {
	testutils.TearDownBadger(suite.db, suite.dir)
}

func (suite *ImageStoreTestSuite) TestImages() {
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

	for _, d := range cves {
		suite.NoError(suite.store.Upsert(d))
	}

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
