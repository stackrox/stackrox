package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	imageComponentSAC = sac.ForResource(resources.Image)
	nodeComponentSAC  = sac.ForResource(resources.Node)

	imageComponentSACFilter filtered.Filter
	combinedFilter          filtered.Filter
	once                    sync.Once
)

// GetSACFilter returns the sac filter for component ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		if features.HostScanning.Enabled() {
			combinedFilter, err = dackbox.NewSharedObjectSACFilter(
				dackbox.WithNode(nodeComponentSAC, dackbox.NodeComponentSACTransform, dackbox.ComponentToNodeExistenceTransformation),
				dackbox.WithImage(imageComponentSAC, dackbox.ImageComponentSACTransform, dackbox.ComponentToImageExistenceTransformation),
				dackbox.WithSharedObjectAccess(storage.Access_READ_ACCESS),
			)
			utils.Must(err)
		} else {
			var err error
			imageComponentSACFilter, err = filtered.NewSACFilter(
				filtered.WithResourceHelper(imageComponentSAC),
				filtered.WithScopeTransform(dackbox.ImageComponentSACTransform),
				filtered.WithReadAccess(),
			)
			utils.Must(err)
		}
	})
	if features.HostScanning.Enabled() {
		return combinedFilter
	}
	return imageComponentSACFilter
}
