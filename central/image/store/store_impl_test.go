package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
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
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *ImageStoreTestSuite) TestImages() {
	images := []*v1.Image{
		{
			Id: "sha1",
			Name: &v1.ImageName{
				FullName: "name1",
			},
		},
		{
			Id: "sha2",
			Name: &v1.ImageName{
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
		input    *v1.Image
		expected *v1.ListImage
	}{
		{
			input: &v1.Image{
				Id: "sha",
				Name: &v1.ImageName{
					FullName: "name",
				},
			},
			expected: &v1.ListImage{
				Id:   "sha",
				Name: "name",
			},
		},
		{
			input: &v1.Image{
				Id: "sha",
				Name: &v1.ImageName{
					FullName: "name",
				},
				Metadata: &v1.ImageMetadata{
					V1: &v1.V1Metadata{
						Created: ts,
					},
				},
				Scan: &v1.ImageScan{
					Components: []*v1.ImageScanComponent{
						{
							Vulns: []*v1.Vulnerability{
								{},
							},
						},
						{
							Vulns: []*v1.Vulnerability{
								{},
								{
									SetFixedBy: &v1.Vulnerability_FixedBy{
										FixedBy: "hi",
									},
								},
							},
						},
					},
				},
			},
			expected: &v1.ListImage{
				Id:      "sha",
				Name:    "name",
				Created: ts,
				SetComponents: &v1.ListImage_Components{
					Components: 2,
				},
				SetCves: &v1.ListImage_Cves{
					Cves: 3,
				},
				SetFixable: &v1.ListImage_FixableCves{
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
