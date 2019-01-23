package aggregation

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// FilterStandards filters the passed standards by the values
// TODO(cgorman) remove these for a real search implementation
func FilterStandards(standards []*v1.ComplianceStandardMetadata, values []string) []string {
	if len(values) == 0 {
		standardIDs := make([]string, 0, len(standards))
		for _, s := range standards {
			standardIDs = append(standardIDs, s.GetId())
		}
		return standardIDs
	}
	var filteredStandards []string
	for _, standard := range standards {
		standardLower := strings.ToLower(standard.GetName())
		for _, v := range values {
			if strings.HasPrefix(standardLower, strings.ToLower(v)) {
				filteredStandards = append(filteredStandards, standard.GetId())
				break
			}
		}
	}
	return filteredStandards
}

// FilterClusters filters the passed clusters by the values
func FilterClusters(clusters []*storage.Cluster, values []string) []string {
	if len(values) == 0 {
		clusterIDs := make([]string, 0, len(clusters))
		for _, s := range clusters {
			clusterIDs = append(clusterIDs, s.GetId())
		}
		return clusterIDs
	}
	var filteredClusters []string
	for _, cluster := range clusters {
		clusterLower := strings.ToLower(cluster.GetName())
		for _, v := range values {
			if strings.HasPrefix(clusterLower, strings.ToLower(v)) {
				filteredClusters = append(filteredClusters, cluster.GetId())
				break
			}
		}
	}
	return filteredClusters
}
