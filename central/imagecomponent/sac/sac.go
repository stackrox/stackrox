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
	imageComponentSAC = sac.ForResource(resources.Image)
	nodeComponentSAC  = sac.ForResource(resources.Node)

	combinedFilter filtered.Filter
	once           sync.Once
)

// GetSACFilter returns the sac filter for component ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		combinedFilter, err = dackbox.NewSharedObjectSACFilter(
			dackbox.WithNode(nodeComponentSAC, dackbox.NodeComponentSACTransform, dackbox.ComponentToNodeExistenceTransformation),
			dackbox.WithImage(imageComponentSAC, dackbox.ImageComponentSACTransform, dackbox.ComponentToImageExistenceTransformation),
			dackbox.WithSharedObjectAccess(storage.Access_READ_ACCESS),
		)
		utils.CrashOnError(err)
	})
	return combinedFilter
}
