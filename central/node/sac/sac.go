package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	nodeSAC = sac.ForResource(resources.Node)

	nodeSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(nodeSAC),
		filtered.WithScopeTransform(dackbox.NodeSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for node ids.
func GetSACFilter() filtered.Filter {
	return nodeSACFilter
}
