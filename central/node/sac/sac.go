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
	nodeSAC = sac.ForResource(resources.Node)

	nodeSACFilter filtered.Filter
	once          sync.Once
)

// GetSACFilter returns the sac filter for node ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		nodeSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(nodeSAC),
			filtered.WithScopeTransform(dackbox.NodeSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)
	})
	return nodeSACFilter
}
