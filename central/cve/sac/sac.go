package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	imageCVESAC   = sac.ForResource(resources.Image)
	nodeCVESAC    = sac.ForResource(resources.Node)
	clusterCVESAC = sac.ForResource(resources.Cluster)

	combinedFilter filtered.Filter
	once           sync.Once
)

// GetSACFilter returns the sac filters for reading cve ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		combinedFilter, err = dackbox.NewSharedObjectSACFilter(
			dackbox.WithNode(nodeCVESAC, dackbox.NodeVulnSACTransform, dackbox.CVEToNodeExistenceTransformation),
			dackbox.WithImage(imageCVESAC, dackbox.ImageVulnSACTransform, dackbox.CVEToImageExistenceTransformation),
			dackbox.WithCluster(clusterCVESAC, dackbox.ClusterVulnSACTransform, dackbox.CVEToClusterExistenceTransformation),
			dackbox.WithSharedObjectAccess(storage.Access_READ_ACCESS),
		)
		utils.CrashOnError(err)
	})
	return combinedFilter
}
