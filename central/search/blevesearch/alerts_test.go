package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestAlertSearch(t *testing.T) {
	suite.Run(t, new(AlertTestSuite))
}

type AlertTestSuite struct {
	suite.Suite
	*Indexer
}

func (suite *AlertTestSuite) SetupSuite() {
	indexer, err := NewTmpIndexer()
	suite.Require().NoError(err)

	suite.Indexer = indexer
	suite.NoError(suite.Indexer.AddAlert(fixtures.GetAlert()))
}

func (suite *AlertTestSuite) TeardownSuite() {
	suite.Indexer.Close()
}

func (suite *AlertTestSuite) TestScopeToAlertQuery() {
	// Test just cluster
	scope := &v1.Scope{
		Cluster: "prod cluster",
	}
	results, err := runQuery(scopeToAlertQuery(scope), suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToAlertQuery(scope), suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "prod cluster",
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToAlertQuery(scope), suite.alertIndex)
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
	results, err = runQuery(scopeToAlertQuery(scope), suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "blah cluster",
		Namespace: "stackrox",
	}
	results, err = runQuery(scopeToAlertQuery(scope), suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 0)
}
