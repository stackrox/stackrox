package resolvers

import (
	"context"
	"fmt"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComplianceResolver(t *testing.T) {
	suite.Run(t, new(ComplianceResolverTestSuite))
}

type ComplianceResolverTestSuite struct {
	suite.Suite
}

func getResultsAndDomains(rowCount int, collapseBy storage.ComplianceAggregation_Scope) ([]*storage.ComplianceAggregation_Result, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain) {
	testResults := make([]*storage.ComplianceAggregation_Result, rowCount*2)
	for i := 0; i < rowCount*2; i += 2 {
		// Create two results per row so tests can make sure collapsing works correctly
		testResults[i] = &storage.ComplianceAggregation_Result{
			AggregationKeys: []*storage.ComplianceAggregation_AggregationKey{
				{
					Scope: collapseBy,
					Id:    fmt.Sprintf("%d", i),
				},
			},
		}
		testResults[i+1] = &storage.ComplianceAggregation_Result{
			AggregationKeys: []*storage.ComplianceAggregation_AggregationKey{
				{
					Scope: collapseBy,
					Id:    fmt.Sprintf("%d", i),
				},
			},
		}
	}
	testDomainMap := make(map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, len(testResults))
	for _, result := range testResults {
		testDomainMap[result] = &storage.ComplianceDomain{}
	}
	return testResults, testDomainMap
}

func (s *ComplianceResolverTestSuite) TestTruncatesAggregationResults() {
	testCollapseBy := storage.ComplianceAggregation_CLUSTER
	testResults, testDomainMap := getResultsAndDomains(aggregationLimit+1, testCollapseBy)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, testCollapseBy)
	s.NotEmpty(errorMessage)
	// We should have two results/collapseBy so these should contain aggregationLimit*2 results
	s.Len(truncatedResults, aggregationLimit*2)
	s.Len(truncatedDomainMap, aggregationLimit*2)
}

func (s *ComplianceResolverTestSuite) TestDoesNotTruncateUnknownCollapseBy() {
	testCollapseBy := storage.ComplianceAggregation_UNKNOWN
	expectedLen := aggregationLimit + 1
	testResults, testDomainMap := getResultsAndDomains(expectedLen, testCollapseBy)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, testCollapseBy)
	s.Empty(errorMessage)
	protoassert.SlicesEqual(s.T(), testResults, truncatedResults)
	protoassert.MapEqual(s.T(), testDomainMap, truncatedDomainMap)
}

func (s *ComplianceResolverTestSuite) TestDoesNotTruncateInvalidCollapseBy() {
	expectedLen := aggregationLimit + 1
	testResults, testDomainMap := getResultsAndDomains(expectedLen, storage.ComplianceAggregation_CLUSTER)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, storage.ComplianceAggregation_NAMESPACE)
	s.Empty(errorMessage)
	protoassert.SlicesEqual(s.T(), testResults, truncatedResults)
	protoassert.MapEqual(s.T(), testDomainMap, truncatedDomainMap)
}

func (s *ComplianceResolverTestSuite) TestDoesNotTruncateShortResults() {
	testCollapseBy := storage.ComplianceAggregation_NAMESPACE
	expectedLen := aggregationLimit - 1
	testResults, testDomainMap := getResultsAndDomains(expectedLen, testCollapseBy)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, testCollapseBy)
	s.Empty(errorMessage)
	protoassert.SlicesEqual(s.T(), testResults, truncatedResults)
	protoassert.MapEqual(s.T(), testDomainMap, truncatedDomainMap)
}

func (s *ComplianceResolverTestSuite) TestTruncateEmptyResults() {
	testCollapseBy := storage.ComplianceAggregation_NAMESPACE
	testResults, testDomainMap := getResultsAndDomains(0, testCollapseBy)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, testCollapseBy)
	s.Empty(errorMessage)
	protoassert.SlicesEqual(s.T(), testResults, truncatedResults)
	protoassert.MapEqual(s.T(), testDomainMap, truncatedDomainMap)
}

func TestComplianceClusters(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	clusterStore := clusterMocks.NewMockDataStore(mockCtrl)
	mainResolver := &Resolver{ClusterDataStore: clusterStore}

	clusterStore.EXPECT().
		SearchRawClusters(gomock.Any(), gomock.Any()).
		Times(1).
		Return(
			[]*storage.Cluster{
				{Id: fixtureconsts.Cluster1, Name: "Cluster 1"},
				{Id: fixtureconsts.Cluster2, Name: "Cluster 2"},
			},
			nil,
		)

	identity := authnMocks.NewMockIdentity(mockCtrl)
	identity.EXPECT().Permissions().Times(1).Return(
		map[string]storage.Access{
			resources.Compliance.String(): storage.Access_READ_ACCESS,
		},
	)

	ctx := sac.WithAllAccess(context.Background())
	ctx = authn.ContextWithIdentity(ctx, identity, t)

	query := PaginatedQuery{}

	fetchedClusterResolvers, err := mainResolver.ComplianceClusters(ctx, query)
	assert.NoError(t, err)

	fetchedScopeObjects := make([]*v1.ScopeObject, 0, len(fetchedClusterResolvers))
	for _, objectResolver := range fetchedClusterResolvers {
		if objectResolver == nil {
			continue
		}
		fetchedScopeObjects = append(fetchedScopeObjects, objectResolver.data)
	}

	expectedScopeObjects := []*v1.ScopeObject{
		{Id: fixtureconsts.Cluster1, Name: "Cluster 1"},
		{Id: fixtureconsts.Cluster2, Name: "Cluster 2"},
	}

	protoassert.ElementsMatch(t, expectedScopeObjects, fetchedScopeObjects)
}
