package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	deploymentSAC = sac.ForResource(resources.Deployment)

	imageSACFilter = filtered.MustCreateNewSACFilter(
		filtered.WithResourceHelper(deploymentSAC),
		filtered.WithScopeTransform(dackbox.DeploymentSACTransform),
		filtered.WithReadAccess())
)

// GetSACFilter returns the sac filter for image ids.
func GetSACFilter() filtered.Filter {
	return imageSACFilter
}
