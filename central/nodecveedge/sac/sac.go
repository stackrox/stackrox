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

	nodeCVEEdgeSACFilter filtered.Filter
	once                 sync.Once
)

// GetSACFilter returns the sac filter for node cve edge ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		nodeCVEEdgeSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(nodeSAC),
			filtered.WithScopeTransform(dackbox.NodeCVEEdgeSACTransform),
			filtered.WithReadAccess(),
		)
		utils.Must(err)
	})
	return nodeCVEEdgeSACFilter
}
