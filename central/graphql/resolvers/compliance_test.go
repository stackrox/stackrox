package resolvers

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
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
	s.Equal(testResults, truncatedResults)
	s.Equal(testDomainMap, truncatedDomainMap)
}

func (s *ComplianceResolverTestSuite) TestDoesNotTruncateInvalidCollapseBy() {
	expectedLen := aggregationLimit + 1
	testResults, testDomainMap := getResultsAndDomains(expectedLen, storage.ComplianceAggregation_CLUSTER)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, storage.ComplianceAggregation_NAMESPACE)
	s.Empty(errorMessage)
	s.Equal(testResults, truncatedResults)
	s.Equal(testDomainMap, truncatedDomainMap)
}

func (s *ComplianceResolverTestSuite) TestDoesNotTruncateShortResults() {
	testCollapseBy := storage.ComplianceAggregation_NAMESPACE
	expectedLen := aggregationLimit - 1
	testResults, testDomainMap := getResultsAndDomains(expectedLen, testCollapseBy)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, testCollapseBy)
	s.Empty(errorMessage)
	s.Equal(testResults, truncatedResults)
	s.Equal(testDomainMap, truncatedDomainMap)
}

func (s *ComplianceResolverTestSuite) TestTruncateEmptyResults() {
	testCollapseBy := storage.ComplianceAggregation_NAMESPACE
	testResults, testDomainMap := getResultsAndDomains(0, testCollapseBy)

	truncatedResults, truncatedDomainMap, errorMessage := truncateResults(testResults, testDomainMap, testCollapseBy)
	s.Empty(errorMessage)
	s.Equal(testResults, truncatedResults)
	s.Equal(testDomainMap, truncatedDomainMap)
}
