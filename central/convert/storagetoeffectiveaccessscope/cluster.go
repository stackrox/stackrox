package storagetoeffectiveaccessscope

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

func Clusters(clusters []*storage.Cluster) []effectiveaccessscope.Cluster {
	if clusters == nil {
		return nil
	}
	result := make([]effectiveaccessscope.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		result = append(result, cluster)
	}
	return result
}
