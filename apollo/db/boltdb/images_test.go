package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltImages(t *testing.T) {
	suite.Run(t, new(BoltImagesTestSuite))
}

type BoltImagesTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltImagesTestSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := MakeBoltDB(tmpDir)
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db.(*BoltDB)
}

func (suite *BoltImagesTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltImagesTestSuite) TestImages() {
	image1 := &v1.Image{
		Sha:      "sha1",
		Registry: "docker.io",
	}
	err := suite.AddImage(image1)
	suite.Nil(err)

	image2 := &v1.Image{
		Sha:      "sha2",
		Registry: "stackrox.io",
	}
	err = suite.AddImage(image2)
	suite.Nil(err)
	// Get all alerts
	images, err := suite.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Image{image1, image2}, images)

	image1.Registry = "stackrox.io"
	err = suite.UpdateImage(image1)
	suite.Nil(err)
	images, err = suite.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Image{image1, image2}, images)

	err = suite.RemoveImage(image1.Sha)
	suite.Nil(err)
	images, err = suite.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Image{image2}, images)
}
