package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
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
