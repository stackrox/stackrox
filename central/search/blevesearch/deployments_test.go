package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentSearch(t *testing.T) {
	suite.Run(t, new(DeploymentTestSuite))
}

type DeploymentTestSuite struct {
	suite.Suite
	*Indexer
}

func (suite *DeploymentTestSuite) SetupSuite() {
	indexer, err := NewIndexer()
	suite.Require().NoError(err)

	suite.Indexer = indexer
	alert := fixtures.GetAlert()
	deployment := alert.GetDeployment()
	suite.NoError(suite.Indexer.AddDeployment(deployment))
}

func (suite *DeploymentTestSuite) TeardownSuite() {
	suite.Indexer.Close()
}

func (suite *DeploymentTestSuite) TestScopeToDeploymentsQuery() {
	// Test just cluster
	scope := &v1.Scope{
		Cluster: "prod cluster",
	}
	results, err := runQuery(scopeToDeploymentQuery(scope), suite.deploymentIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToDeploymentQuery(scope), suite.deploymentIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "prod cluster",
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToDeploymentQuery(scope), suite.deploymentIndex)
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
	results, err = runQuery(scopeToDeploymentQuery(scope), suite.deploymentIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "blah cluster",
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToDeploymentQuery(scope), suite.deploymentIndex)
	suite.NoError(err)
	suite.Len(results, 0)
}
