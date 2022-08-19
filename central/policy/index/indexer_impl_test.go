package index

import (
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/stackrox/rox/central/globalindex"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	fakeID       = "FAKEID"
	fakeSeverity = storage.Severity_HIGH_SEVERITY
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
	secondPolicy.LifecycleStages = []storage.LifecycleStage{storage.LifecycleStage_DEPLOY}
	suite.NoError(suite.indexer.AddPolicies([]*storage.Policy{secondPolicy}))
}

func (suite *PolicyIndexTestSuite) TestPolicySearch() {
	cases := []struct {
		name        string
		q           *v1.Query
		expectedIDs []string
		expectedErr bool
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
			q:           search.NewQueryBuilder().AddStrings(search.Severity, "low").ProtoQuery(),
			expectedIDs: []string{fixtures.GetPolicy().GetId()},
		},
		{
			name:        "Invalid query for policy",
			q:           search.NewQueryBuilder().AddStrings(search.DeploymentName, "fake").ProtoQuery(),
			expectedIDs: []string{},
		},
		{
			name:        "Lifecycle stage prefix",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "deplo").ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Lifecycle stage exact match doesn't match",
			q:           search.NewQueryBuilder().AddExactMatches(search.LifecycleStage, "deplo").ProtoQuery(),
			expectedErr: true,
		},
		{
			name:        "Lifecycle stage prefix with full string",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "deploy").ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Lifecycle stage exact match matches",
			q:           search.NewQueryBuilder().AddExactMatches(search.LifecycleStage, "deploy").ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Lifecycle stage regex no match",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "r/asab").ProtoQuery(),
			expectedErr: true,
		},
		{
			name:        "Lifecycle stage regex matches one",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "r/dep.*").ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Lifecycle stage regex matches all",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "r/.*").ProtoQuery(),
			expectedIDs: []string{fakeID, fixtures.GetPolicy().GetId()},
		},
		{
			name:        "Lifecycle stage with negation",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "!deploy").ProtoQuery(),
			expectedIDs: []string{fixtures.GetPolicy().GetId()},
		},
		{
			name:        "Lifecycle stage with negated regex (matches one)",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "!r/depl").ProtoQuery(),
			expectedIDs: []string{fixtures.GetPolicy().GetId()},
		},
		{
			name:        "Lifecycle stage with negated regex (but doesn't match)",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "!r/blah").ProtoQuery(),
			expectedIDs: []string{fixtures.GetPolicy().GetId(), fakeID},
		},
		{
			name:        "Lifecycle stage with negated regex (matches both)",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "!r/.*").ProtoQuery(),
			expectedErr: true,
		},
		{
			name:        "Lifecycle stage with negated exact match",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "!\"depl\"").ProtoQuery(),
			expectedIDs: []string{fixtures.GetPolicy().GetId(), fakeID},
		},
		{
			name:        "Lifecycle stage with negated exact match (but matches)",
			q:           search.NewQueryBuilder().AddStrings(search.LifecycleStage, "!\"deploy\"").ProtoQuery(),
			expectedIDs: []string{fixtures.GetPolicy().GetId()},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			results, err := suite.indexer.Search(c.q)
			if c.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.ID)
			}
			assert.ElementsMatch(t, resultIDs, c.expectedIDs)
		})
	}
}

func (suite *PolicyIndexTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
}
