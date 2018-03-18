package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestImageSearch(t *testing.T) {
	suite.Run(t, new(ImageTestSuite))
}

type ImageTestSuite struct {
	suite.Suite
	*Indexer
}

func (suite *ImageTestSuite) SetupSuite() {
	indexer, err := NewIndexer()
	suite.Require().NoError(err)

	suite.Indexer = indexer
	suite.NoError(suite.Indexer.AddDeployment(fixtures.GetAlert().GetDeployment()))

	for _, c := range fixtures.GetAlert().GetDeployment().GetContainers() {
		suite.Indexer.AddImage(c.GetImage())
	}
}

func (suite *ImageTestSuite) TeardownSuite() {
	suite.Indexer.Close()
}

func (suite *ImageTestSuite) TestSearchImages() {
	// Test just cluster -> should give all images
	request := &v1.SearchRequest{
		Scopes: []*v1.Scope{
			{
				Cluster: "prod cluster",
			},
		},
	}

	results, err := suite.SearchImages(request)
	suite.NoError(err)
	suite.Len(results, 2)

	// Test both scopes and fields defined
	request = &v1.SearchRequest{
		Scopes: []*v1.Scope{
			{
				Cluster: "prod cluster",
			},
		},
		Fields: map[string]*v1.SearchRequest_Values{
			"image.name.registry": {
				Values: []string{"docker.io", "stackrox.io"},
			},
		},
	}

	results, err = suite.SearchImages(request)
	suite.NoError(err)
	suite.Len(results, 2)

	// Test No scopes and should image

	// Test Failure

}
