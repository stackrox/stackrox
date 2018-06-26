package index

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/globalindex"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	bleveHelpers "bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve"
	"github.com/stretchr/testify/suite"
)

func TestAlertIndex(t *testing.T) {
	suite.Run(t, new(PolicyIndexTestSuite))
}

type PolicyIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer Indexer
}

func (suite *PolicyIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

	alert := fixtures.GetAlert()
	policy := alert.GetPolicy()
	suite.NoError(suite.indexer.AddPolicy(policy))
}

func (suite *PolicyIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *PolicyIndexTestSuite) TestScopeToPolicyQuery() {
	// Test just cluster
	scope := &v1.Scope{
		Cluster: "prod cluster",
	}
	results, err := bleveHelpers.RunQuery(ScopeToPolicyQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToPolicyQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "prod cluster",
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToPolicyQuery(scope), suite.bleveIndex)
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
	results, err = bleveHelpers.RunQuery(ScopeToPolicyQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "blah cluster",
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToPolicyQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 0)
}
