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
		ca := &storage.ComplianceAggregation_AggregationKey{}
		ca.SetScope(collapseBy)
		ca.SetId(fmt.Sprintf("%d", i))
		cr := &storage.ComplianceAggregation_Result{}
		cr.SetAggregationKeys([]*storage.ComplianceAggregation_AggregationKey{
			ca,
		})
		testResults[i] = cr
		ca2 := &storage.ComplianceAggregation_AggregationKey{}
		ca2.SetScope(collapseBy)
		ca2.SetId(fmt.Sprintf("%d", i))
		cr2 := &storage.ComplianceAggregation_Result{}
		cr2.SetAggregationKeys([]*storage.ComplianceAggregation_AggregationKey{
			ca2,
		})
		testResults[i+1] = cr2
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

	cluster := &storage.Cluster{}
	cluster.SetId(fixtureconsts.Cluster1)
	cluster.SetName("Cluster 1")
	cluster2 := &storage.Cluster{}
	cluster2.SetId(fixtureconsts.Cluster2)
	cluster2.SetName("Cluster 2")
	clusterStore.EXPECT().
		SearchRawClusters(gomock.Any(), gomock.Any()).
		Times(1).
		Return(
			[]*storage.Cluster{
				cluster,
				cluster2,
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

	so := &v1.ScopeObject{}
	so.SetId(fixtureconsts.Cluster1)
	so.SetName("Cluster 1")
	so2 := &v1.ScopeObject{}
	so2.SetId(fixtureconsts.Cluster2)
	so2.SetName("Cluster 2")
	expectedScopeObjects := []*v1.ScopeObject{
		so,
		so2,
	}

	protoassert.ElementsMatch(t, expectedScopeObjects, fetchedScopeObjects)
}
