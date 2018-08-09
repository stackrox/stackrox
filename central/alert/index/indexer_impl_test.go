package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	bleveHelpers "github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestAlertIndex(t *testing.T) {
	suite.Run(t, new(AlertIndexTestSuite))
}

type AlertIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer Indexer
}

func (suite *AlertIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

	suite.NoError(suite.indexer.AddAlert(fixtures.GetAlert()))
}

func (suite *AlertIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *AlertIndexTestSuite) TestScopeToAlertQuery() {
	// Test just cluster
	scope := &v1.Scope{
		Cluster: "prod cluster",
	}
	results, err := bleveHelpers.RunQuery(ScopeToAlertQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToAlertQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "prod cluster",
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToAlertQuery(scope), suite.bleveIndex)
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
	results, err = bleveHelpers.RunQuery(ScopeToAlertQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 1)

	scope = &v1.Scope{
		Cluster:   "blah cluster",
		Namespace: "stackrox",
	}
	results, err = bleveHelpers.RunQuery(ScopeToAlertQuery(scope), suite.bleveIndex)
	suite.NoError(err)
	suite.Len(results, 0)
}

func (suite *AlertIndexTestSuite) TestDefaultStaleness() {

	var cases = []struct {
		name               string
		values             []string
		expectedViolations int
	}{
		{
			name:               "no stale field",
			values:             []string{},
			expectedViolations: 1,
		},
		{
			name:               "stale = false",
			values:             []string{"false"},
			expectedViolations: 1,
		},
		{
			name:               "stale = true",
			values:             []string{"true"},
			expectedViolations: 0,
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			alerts, err := suite.indexer.SearchAlerts(&v1.ParsedSearchRequest{
				Fields: map[string]*v1.ParsedSearchRequest_Values{
					search.Stale: {
						Values: c.values,
					},
				},
			})
			assert.NoError(t, err)
			assert.Len(t, alerts, c.expectedViolations)
		})
	}
}
