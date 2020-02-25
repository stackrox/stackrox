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

	componentCVEEdgeSACFilter filtered.Filter
	once                      sync.Once
)

// GetSACFilter returns the sac filter for componentCVEEdge ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		componentCVEEdgeSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageSAC),
		)
		utils.Must(err)
	})
	return componentCVEEdgeSACFilter
}
