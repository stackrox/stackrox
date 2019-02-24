package store

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestImageStore(t *testing.T) {
	suite.Run(t, new(ImageStoreTestSuite))
}

type ImageStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *ImageStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *ImageStoreTestSuite) TearDownSuite() {
	testutils.TearDownDB(suite.db)
}

func (suite *ImageStoreTestSuite) TestImages() {
	images := []*storage.Image{
		{
			Id: "sha1",
			Name: &storage.ImageName{
				FullName: "name1",
			},
		},
		{
			Id: "sha2",
			Name: &storage.ImageName{
				FullName: "name2",
			},
		},
	}

	// Test Add
	for _, d := range images {
		suite.NoError(suite.store.UpsertImage(d))
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
		suite.NoError(suite.store.UpsertImage(d))
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

func (suite *ImageStoreTestSuite) TestConvertImagesToListImages() {
	ts := timestamp.TimestampNow()
	var cases = []struct {
		input    *storage.Image
		expected *storage.ListImage
	}{
		{
			input: &storage.Image{
				Id: "sha",
				Name: &storage.ImageName{
					FullName: "name",
				},
			},
			expected: &storage.ListImage{
				Id:   "sha",
				Name: "name",
			},
		},
		{
			input: &storage.Image{
				Id: "sha",
				Name: &storage.ImageName{
					FullName: "name",
				},
				Metadata: &storage.ImageMetadata{
					V1: &storage.V1Metadata{
						Created: ts,
					},
				},
				Scan: &storage.ImageScan{
					Components: []*storage.ImageScanComponent{
						{
							Vulns: []*storage.Vulnerability{
								{},
							},
						},
						{
							Vulns: []*storage.Vulnerability{
								{},
								{
									SetFixedBy: &storage.Vulnerability_FixedBy{
										FixedBy: "hi",
									},
								},
							},
						},
					},
				},
			},
			expected: &storage.ListImage{
				Id:      "sha",
				Name:    "name",
				Created: ts,
				SetComponents: &storage.ListImage_Components{
					Components: 2,
				},
				SetCves: &storage.ListImage_Cves{
					Cves: 3,
				},
				SetFixable: &storage.ListImage_FixableCves{
					FixableCves: 1,
				},
			},
		},
	}
	for _, c := range cases {
		suite.T().Run(c.input.GetName().GetFullName(), func(t *testing.T) {
			assert.Equal(t, c.expected, convertImageToListImage(c.input))
		})
	}
}
