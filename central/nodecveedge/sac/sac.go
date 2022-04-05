package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
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
