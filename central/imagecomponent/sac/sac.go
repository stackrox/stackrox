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

	imageComponentSACFilter filtered.Filter
	once                    sync.Once
)

// GetSACFilter returns the sac filter for image component ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		imageComponentSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageComponentSAC),
			filtered.WithScopeTransform(dackbox.ComponentSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)
	})
	return imageComponentSACFilter
}
