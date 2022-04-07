package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	imageSAC = sac.ForResource(resources.Image)
	nodeSAC  = sac.ForResource(resources.Node)

	componentCVEEdgeSACFilter = filtered.NewEdgeSourceFilter(
		dackbox.MustCreateNewSharedObjectSACFilter(
			dackbox.WithNode(nodeSAC, dackbox.ComponentToNodeBucketPath),
			dackbox.WithImage(imageSAC, dackbox.ComponentToImageBucketPath),
			dackbox.WithSharedObjectAccess(storage.Access_READ_ACCESS),
		),
	)
)

// GetSACFilter returns the sac filter for componentCVEEdge ids.
func GetSACFilter() filtered.Filter {
	return componentCVEEdgeSACFilter
}
