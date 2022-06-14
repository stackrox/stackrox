package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	clusterSAC = sac.ForResource(resources.Cluster)

	clusterSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(clusterSAC),
		filtered.WithScopeTransform(dackbox.ClusterSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for cluster ids.
func GetSACFilter() filtered.Filter {
	return clusterSACFilter
}
