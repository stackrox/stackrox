package testutils

import (
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/sac"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/suite"
)

// ValidateSACSearchResultDistribution checks whether the obtained result distribution map has the
// same result distribution as the expected one.
func ValidateSACSearchResultDistribution(s *suite.Suite, expected, obtained map[string]map[string]int) {
	s.Equal(len(expected), len(obtained), "unexpected cluster count in result")
	for clusterID, clusterMap := range expected {
		_, clusterFound := obtained[clusterID]
		s.True(clusterFound)
		if clusterFound {
			for namespace, count := range clusterMap {
				_, namespaceFound := obtained[clusterID][namespace]
				s.True(namespaceFound)
				s.Equalf(count, obtained[clusterID][namespace], "unexpected count for cluster %s and namespace %s", clusterID, namespace)
			}
		}
	}
}

// CountResultsPerClusterAndNamespace builds a result distribution map from the search output of a test,
// counting the results per cluster and namespace.
func CountResultsPerClusterAndNamespace(_ *testing.T, searchResults []searchPkg.Result, optionsMap searchPkg.OptionsMap) map[string]map[string]int {
	resultDistribution := make(map[string]map[string]int, 0)
	clusterIDField, _ := optionsMap.Get(searchPkg.ClusterID.String())
	namespaceField, _ := optionsMap.Get(searchPkg.Namespace.String())
	for _, result := range searchResults {
		var clusterID string
		var namespace string
		for k, v := range result.Matches {
			if k == clusterIDField.GetFieldPath() {
				clusterID = v[0]
			}
			if k == namespaceField.GetFieldPath() {
				namespace = v[0]
			}
		}
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

// CountSearchResultsPerClusterAndNamespace builds a result distribution map from the search output of a test,
// counting the results per cluster and namespace.
func CountSearchResultsPerClusterAndNamespace(_ *testing.T, results []*v1.SearchResult, optionsMap searchPkg.OptionsMap) map[string]map[string]int {
	resultDistribution := make(map[string]map[string]int, 0)
	clusterIDField, _ := optionsMap.Get(searchPkg.ClusterID.String())
	namespaceField, _ := optionsMap.Get(searchPkg.Namespace.String())
	for _, result := range results {
		var clusterID string
		var namespace string
		for k, v := range result.GetFieldToMatches() {
			if k == clusterIDField.GetFieldPath() {
				if v != nil && len(v.Values) > 0 {
					clusterID = v.Values[0]
				}
			}
			if k == namespaceField.GetFieldPath() {
				if v != nil && len(v.Values) > 0 {
					namespace = v.Values[0]
				}
			}
		}
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
