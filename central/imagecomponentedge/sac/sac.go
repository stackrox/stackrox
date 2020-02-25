package sac

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	imageSAC = sac.ForResource(resources.Image)

	imageComponentEdgeSACFilter filtered.Filter
	once                        sync.Once
)

// GetSACFilter returns the sac filter for image component edge ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		imageComponentEdgeSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageSAC),
		)
		utils.Must(err)
	})
	return imageComponentEdgeSACFilter
}
