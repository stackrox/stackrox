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
	imageCVESAC   = sac.ForResource(resources.Image)
	nodeCVESAC    = sac.ForResource(resources.Node)
	clusterCVESAC = sac.ForResource(resources.Cluster)

	clusterCVESACFilter filtered.Filter
	imageCVESACFilter   filtered.Filter
	nodeCVESACFilter    filtered.Filter
	once                sync.Once
)

// GetSACFilters returns the sac filters for reading cve ids.
func GetSACFilters() []filtered.Filter {
	once.Do(func() {
		var err error
		imageCVESACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageCVESAC),
			filtered.WithScopeTransform(dackbox.ImageVulnSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)

		nodeCVESACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(nodeCVESAC),
			filtered.WithScopeTransform(dackbox.NodeVulnSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)

		clusterCVESACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(clusterCVESAC),
			filtered.WithScopeTransform(dackbox.ClusterVulnSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)
	})
	return []filtered.Filter{imageCVESACFilter, nodeCVESACFilter, clusterCVESACFilter}
}
