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

func (suite *ImageStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *ImageStoreTestSuite) TestImages() {
	images := []*v1.Image{
		{
			Name: &v1.ImageName{
				Sha:      "sha1",
				FullName: "name1",
			},
		},
		{
			Name: &v1.ImageName{
				Sha:      "sha2",
				FullName: "name2",
			},
		},
	}

	// Test Add
	for _, d := range images {
		suite.NoError(suite.store.UpsertImage(d))
	}

	for _, d := range images {
		got, exists, err := suite.store.GetImage(d.GetName().GetSha())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)

		listGot, exists, err := suite.store.ListImage(d.GetName().GetSha())
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
		got, exists, err := suite.store.GetImage(d.GetName().GetSha())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)

		listGot, exists, err := suite.store.ListImage(d.GetName().GetSha())
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
				Name: &v1.ImageName{
					Sha:      "sha",
					FullName: "name",
				},
			},
			expected: &v1.ListImage{
				Sha:  "sha",
				Name: "name",
			},
		},
		{
			input: &v1.Image{
				Name: &v1.ImageName{
					Sha:      "sha",
					FullName: "name",
				},
				Metadata: &v1.ImageMetadata{
					Created: ts,
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
				Sha:     "sha",
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

func (suite *ImageStoreTestSuite) TestShas() {
	sha1 := "sha1"
	sha2 := "sha2"
	regSha1 := "sha3"
	regSha2 := "sha4"

	// Upsert shas
	err := suite.store.UpsertRegistrySha(sha1, regSha1)
	suite.Nil(err)

	err = suite.store.UpsertRegistrySha(sha2, regSha2)
	suite.Nil(err)

	// Get Sha
	retrievedsha, exists, err := suite.store.GetRegistrySha(sha1)
	suite.Nil(err)
	suite.True(exists)
	suite.Equal(regSha1, retrievedsha)

	// Delete sha
	err = suite.store.DeleteRegistrySha("sha1")
	suite.Nil(err)

	// Get sha
	retrievedsha, exists, err = suite.store.GetRegistrySha("sha1")
	suite.Nil(err)
	suite.False(exists)
}
