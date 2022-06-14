package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	imageSAC = sac.ForResource(resources.Image)

	imageSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(imageSAC),
		filtered.WithScopeTransform(dackbox.ImageSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for image ids.
func GetSACFilter() filtered.Filter {
	return imageSACFilter
}
