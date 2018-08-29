package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	fakeID       = "FAKEID"
	fakeSeverity = v1.Severity_HIGH_SEVERITY
)

func TestPolicyIndex(t *testing.T) {
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

	policy := fixtures.GetPolicy()
	suite.NoError(suite.indexer.AddPolicy(policy))

	secondPolicy := fixtures.GetPolicy()
	secondPolicy.Id = fakeID
	secondPolicy.Severity = fakeSeverity
	suite.NoError(suite.indexer.AddPolicies([]*v1.Policy{secondPolicy}))
}

func (suite *PolicyIndexTestSuite) TestPolicySearch() {
	cases := []struct {
		name        string
		q           *v1.Query
		expectedIDs []string
	}{
		{
			name:        "Empty",
			q:           search.EmptyQuery(),
			expectedIDs: []string{fakeID, fixtures.GetPolicy().GetId()},
		},
		{
			name:        "Matching both",
			q:           search.NewQueryBuilder().AddStrings(search.PolicyName, "vulnerable").ProtoQuery(),
			expectedIDs: []string{fakeID, fixtures.GetPolicy().GetId()},
		},
		{
			name:        "Matching severity",
			q:           search.NewQueryBuilder().AddStrings(search.Severity, "l").ProtoQuery(),
			expectedIDs: []string{fixtures.GetPolicy().GetId()},
		},
		{
			name:        "Invalid query for policy",
			q:           search.NewQueryBuilder().AddStrings(search.DeploymentName, "fake").ProtoQuery(),
			expectedIDs: []string{},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			results, err := suite.indexer.SearchPolicies(c.q)
			require.NoError(t, err)
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.ID)
			}
			assert.ElementsMatch(t, resultIDs, c.expectedIDs)
		})
	}
}

func (suite *PolicyIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}
