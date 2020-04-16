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
	clusterCVESAC = sac.ForResource(resources.Cluster)

	clusterCVESACFilter filtered.Filter
	cveSACFilter        filtered.Filter
	once                sync.Once
)

// GetSACFilters returns the sac filters for reading cve ids.
func GetSACFilters() []filtered.Filter {
	once.Do(func() {
		var err error
		cveSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageCVESAC),
			filtered.WithScopeTransform(dackbox.VulnSACTransform),
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
	return []filtered.Filter{cveSACFilter, clusterCVESACFilter}
}
