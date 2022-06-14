package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

var (
	imageCVESAC   = sac.ForResource(resources.Image)
	nodeCVESAC    = sac.ForResource(resources.Node)
	clusterCVESAC = sac.ForResource(resources.Cluster)

	combinedFilter = dackbox.MustCreateNewSharedObjectSACFilter(
		dackbox.WithNode(nodeCVESAC, dackbox.CVEToNodeBucketPath),
		dackbox.WithImage(imageCVESAC, dackbox.CVEToImageBucketPath),
		dackbox.WithCluster(clusterCVESAC, dackbox.CVEToClusterBucketPath),
		dackbox.WithSharedObjectAccess(storage.Access_READ_ACCESS),
	)
)

// GetSACFilter returns the sac filters for reading cve ids.
func GetSACFilter() filtered.Filter {
	return combinedFilter
}
