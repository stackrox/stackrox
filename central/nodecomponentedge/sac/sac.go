package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

var (
	nodeSAC = sac.ForResource(resources.Node)

	nodeComponentEdgeSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(nodeSAC),
		filtered.WithScopeTransform(dackbox.NodeComponentEdgeSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for node component edge ids.
func GetSACFilter() filtered.Filter {
	return nodeComponentEdgeSACFilter
}
