package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	imageSAC = sac.ForResource(resources.Image)

	imageCVEEdgeSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(imageSAC),
		filtered.WithScopeTransform(dackbox.ImageCVEEdgeSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for image cve edge ids.
func GetSACFilter() filtered.Filter {
	return imageCVEEdgeSACFilter
}
