package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	bleveHelpers "github.com/stackrox/rox/pkg/search/blevesearch"
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

	alert := fixtures.GetAlert()
	deployment := alert.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(deployment))
}

func (suite *DeploymentIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *DeploymentIndexTestSuite) TestScopeToDeploymentsQuery() {
	// Test just cluster
	scope := &v1.Scope{
		Cluster: "prod cluster",
	}
	results, err := bleveHelpers.RunQuery(ScopeToDeploymentQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToDeploymentQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "prod cluster",
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToDeploymentQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "prod cluster",
		Namespace: "stackrox",
		Label: &v1.Scope_Label{
			Key:   "com.docker.stack.namespace",
			Value: "prevent",
		},
	}
	results, err = bleveHelpers.RunQuery(ScopeToDeploymentQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "blah cluster",
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToDeploymentQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 0)
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
}
