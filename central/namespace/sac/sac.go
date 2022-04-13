package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

var (
	nsSAC = sac.ForResource(resources.Namespace)

	nsSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(nsSAC),
		filtered.WithScopeTransform(dackbox.NamespaceSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for image ids.
func GetSACFilter() filtered.Filter {
	return nsSACFilter
}
