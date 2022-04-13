package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
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
