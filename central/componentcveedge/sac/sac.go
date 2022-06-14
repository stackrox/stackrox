package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
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
