package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

// ValidateSACSearchResultDistribution checks whether the obtained result distribution map has the
// same result distribution as the expected one.
func ValidateSACSearchResultDistribution(s *suite.Suite, expected, obtained map[string]map[string]int) {
	s.Equal(len(expected), len(obtained), "unexpected cluster count in result")
	for clusterID, clusterMap := range expected {
		_, clusterFound := obtained[clusterID]
		s.Truef(clusterFound, "Cluster %s not found in results", clusterID)
		if clusterFound {
			for namespace, count := range clusterMap {
				_, namespaceFound := obtained[clusterID][namespace]
				s.True(namespaceFound, "Namespace %s not found in cluster %s results", namespace, clusterID)
				s.Equalf(count, obtained[clusterID][namespace], "unexpected count for cluster %s and namespace %s", clusterID, namespace)
			}
		}
	}
}

// AggregateCounts returns the aggregated result count of an expected test result distribution map
func AggregateCounts(_ *testing.T, resultDistribution map[string]map[string]int) int {
	sum := 0
	for _, submap := range resultDistribution {
		for _, count := range submap {
			sum += count
		}
	}
	return sum
}

// CountSearchResultObjectsPerClusterAndNamespace builds a result distribution map from the search output of a test,
// counting the results per cluster and namespace.
func CountSearchResultObjectsPerClusterAndNamespace(_ *testing.T, results []sac.NamespaceScopedObject) map[string]map[string]int {
	resultDistribution := make(map[string]map[string]int, 0)
	for _, result := range results {
		if result == nil {
			continue
		}
		clusterID := result.GetClusterId()
		namespace := result.GetNamespace()
		if _, clusterIDExists := resultDistribution[clusterID]; !clusterIDExists {
			resultDistribution[clusterID] = make(map[string]int, 0)
		}
		if _, namespaceExists := resultDistribution[clusterID][namespace]; !namespaceExists {
			resultDistribution[clusterID][namespace] = 0
		}
		resultDistribution[clusterID][namespace]++
	}
	return resultDistribution
}
