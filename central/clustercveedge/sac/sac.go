package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	clusterSAC = sac.ForResource(resources.Cluster)

	clusterCVEEdgeSACFilter filtered.Filter
	once                    sync.Once
)

// GetSACFilter returns the sac filter for ClusterCVEEdge ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		clusterCVEEdgeSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(clusterSAC),
			filtered.WithScopeTransform(dackbox.ClusterVulnEdgeSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)
	})
	return clusterCVEEdgeSACFilter
}
