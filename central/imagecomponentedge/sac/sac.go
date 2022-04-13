package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

var (
	imageSAC = sac.ForResource(resources.Image)

	imageComponentEdgeSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(imageSAC),
		filtered.WithScopeTransform(dackbox.ImageComponentEdgeSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for image component edge ids.
func GetSACFilter() filtered.Filter {
	return imageComponentEdgeSACFilter
}
