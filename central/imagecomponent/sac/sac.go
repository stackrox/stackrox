package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

var (
	imageComponentSAC = sac.ForResource(resources.Image)
	nodeComponentSAC  = sac.ForResource(resources.Node)

	combinedFilter = dackbox.MustCreateNewSharedObjectSACFilter(
		dackbox.WithNode(nodeComponentSAC, dackbox.ComponentToNodeBucketPath),
		dackbox.WithImage(imageComponentSAC, dackbox.ComponentToImageBucketPath),
		dackbox.WithSharedObjectAccess(storage.Access_READ_ACCESS),
	)
)

// GetSACFilter returns the sac filter for component ids.
func GetSACFilter() filtered.Filter {
	return combinedFilter
}
