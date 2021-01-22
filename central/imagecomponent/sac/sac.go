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
	imageComponentSAC = sac.ForResource(resources.Image)
	nodeComponentSAC  = sac.ForResource(resources.Node)

	imageComponentSACFilter filtered.Filter
	nodeComponentSACFilter  filtered.Filter
	once                    sync.Once
)

// GetSACFilters returns the sac filter for component ids.
func GetSACFilters() []filtered.Filter {
	once.Do(func() {
		var err error
		imageComponentSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageComponentSAC),
			filtered.WithScopeTransform(dackbox.ImageComponentSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)

		nodeComponentSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(nodeComponentSAC),
			filtered.WithScopeTransform(dackbox.NodeComponentSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)
	})
	return []filtered.Filter{imageComponentSACFilter, nodeComponentSACFilter}
}
