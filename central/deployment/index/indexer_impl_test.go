package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentIndex(t *testing.T) {
	suite.Run(t, new(DeploymentIndexTestSuite))
}

type DeploymentIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer Indexer
}

func (suite *DeploymentIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

	deployment := fixtures.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(deployment))

	imageIndexer := index.New(tmpIndex)
	imageIndexer.AddImage(fixtures.GetImage())
}

func (suite *DeploymentIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *DeploymentIndexTestSuite) TestDeploymentsQuery() {
	results, err := suite.indexer.SearchDeployments(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			search.DeploymentName: {
				Values: []string{"nginx"},
			},
		},
	})
	suite.NoError(err)
	suite.Len(results, 1)

	results, err = suite.indexer.SearchDeployments(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			search.DeploymentName: {
				Values: []string{"!nginx"},
			},
		},
	})
	suite.NoError(err)
	suite.Len(results, 0)

	results, err = suite.indexer.SearchDeployments(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			search.DeploymentName: {
				Values: []string{"!nomatch"},
			},
		},
	})
	suite.NoError(err)
	suite.Len(results, 1)

	results, err = suite.indexer.SearchDeployments(&v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			search.DeploymentName: {
				Values: []string{"!nomatch"},
			},
			search.ImageRegistry: {
				Values: []string{"stackrox"},
			},
		},
	})
	suite.NoError(err)
	suite.Len(results, 1)
}
