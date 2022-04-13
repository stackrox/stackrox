package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

var (
	nodeSAC = sac.ForResource(resources.Node)

	nodeCVEEdgeSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(nodeSAC),
		filtered.WithScopeTransform(dackbox.NodeCVEEdgeSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for node cve edge ids.
func GetSACFilter() filtered.Filter {
	return nodeCVEEdgeSACFilter
}
