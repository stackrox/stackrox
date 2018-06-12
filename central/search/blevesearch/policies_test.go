package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestPolicySearch(t *testing.T) {
	suite.Run(t, new(PolicyTestSuite))
}

type PolicyTestSuite struct {
	suite.Suite
	*Indexer
}

func (suite *PolicyTestSuite) SetupSuite() {
	indexer, err := NewTmpIndexer()
	suite.Require().NoError(err)

	suite.Indexer = indexer
	alert := fixtures.GetAlert()
	policy := alert.GetPolicy()
	suite.NoError(suite.Indexer.AddPolicy(policy))
}

func (suite *PolicyTestSuite) TeardownSuite() {
	suite.Indexer.Close()
}

func (suite *PolicyTestSuite) TestScopeToPolicyQuery() {
	// Test just cluster
	scope := &v1.Scope{
		Cluster: "prod cluster",
	}
	results, err := runQuery(scopeToPolicyQuery(scope), suite.globalIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToPolicyQuery(scope), suite.globalIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "prod cluster",
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToPolicyQuery(scope), suite.globalIndex)
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
	results, err = runQuery(scopeToPolicyQuery(scope), suite.globalIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "blah cluster",
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToPolicyQuery(scope), suite.globalIndex)
	suite.NoError(err)
	suite.Len(results, 0)
}
