package index

import (
	"testing"

	deploymentIndex "bitbucket.org/stack-rox/apollo/central/deployment/index"
	"bitbucket.org/stack-rox/apollo/central/globalindex"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/blevesearch/bleve"
	"github.com/stretchr/testify/suite"
)

func TestImageIndex(t *testing.T) {
	suite.Run(t, new(ImageIndexTestSuite))
}

type ImageIndexTestSuite struct {
	suite.Suite

	bleveIndex        bleve.Index
	deploymentIndexer deploymentIndex.Indexer

	indexer Indexer
}

func (suite *ImageIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.deploymentIndexer = deploymentIndex.New(tmpIndex)

	suite.indexer = New(tmpIndex)

	suite.NoError(suite.deploymentIndexer.AddDeployment(fixtures.GetAlert().GetDeployment()))

	for _, c := range fixtures.GetAlert().GetDeployment().GetContainers() {
		suite.indexer.AddImage(c.GetImage())
	}
}

func (suite *ImageIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *ImageIndexTestSuite) TestSearchImages() {
	// Test no fields give us all of the images.
	request := &v1.ParsedSearchRequest{}

	results, err := suite.indexer.SearchImages(request)
	suite.NoError(err)
	suite.Len(results, 2)

	// Test just cluster -> should give all images
	request = &v1.ParsedSearchRequest{
		Scopes: []*v1.Scope{
			{
				Cluster: "prod cluster",
			},
		},
	}

	results, err = suite.indexer.SearchImages(request)
	suite.NoError(err)
	suite.Len(results, 2)

	// Test both scopes and fields defined
	request = &v1.ParsedSearchRequest{
		Scopes: []*v1.Scope{
			{
				Cluster: "prod cluster",
			},
		},
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"image.name.registry": {
				Field: &v1.SearchField{
					FieldPath: "image.name.registry",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"stackrox.io"},
			},
		},
	}

	results, err = suite.indexer.SearchImages(request)
	suite.NoError(err)
	suite.Len(results, 1)

	// Test only fields defined
	request = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"image.name.registry": {
				Field: &v1.SearchField{
					FieldPath: "image.name.registry",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"stackrox.io"},
			},
		},
	}

	results, err = suite.indexer.SearchImages(request)
	suite.NoError(err)
	suite.Len(results, 1)
}
