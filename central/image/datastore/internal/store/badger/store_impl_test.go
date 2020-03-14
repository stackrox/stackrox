package badger

import (
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestImageStore(t *testing.T) {
	suite.Run(t, new(ImageStoreTestSuite))
}

type ImageStoreTestSuite struct {
	suite.Suite

	db  *badger.DB
	dir string

	store store.Store
}

func (suite *ImageStoreTestSuite) SetupSuite() {
	db, dir, err := badgerhelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.dir = dir
	suite.store = New(db, false)
}

func (suite *ImageStoreTestSuite) TearDownSuite() {
	testutils.TearDownBadger(suite.db, suite.dir)
}

func (suite *ImageStoreTestSuite) TestImages() {
	images := []*storage.Image{
		{
			Id: "sha256:sha1",
			Name: &storage.ImageName{
				FullName: "name1",
			},
		},
		{
			Id: "sha256:sha2",
			Name: &storage.ImageName{
				FullName: "name2",
			},
		},
	}

	// Test Add
	for _, d := range images {
		suite.NoError(suite.store.Upsert(d))
	}

	for _, d := range images {
		got, exists, err := suite.store.GetImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)

		listGot, exists, err := suite.store.ListImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(listGot.GetName(), d.GetName().GetFullName())
	}

	// Test Update
	for _, d := range images {
		d.Name.FullName += "1"
	}

	for _, d := range images {
		suite.NoError(suite.store.Upsert(d))
	}

	for _, d := range images {
		got, exists, err := suite.store.GetImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)

		listGot, exists, err := suite.store.ListImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(listGot.GetName(), d.GetName().GetFullName())
	}

	// Test Count
	count, err := suite.store.CountImages()
	suite.NoError(err)
	suite.Equal(len(images), count)
}
