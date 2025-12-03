package images

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
)

// ImageDeploymentOptions defines the deployment options available to search on images
var ImageDeploymentOptions = func() search.OptionsMap {
	var searchCategory v1.SearchCategory
	if features.FlattenImageData.Enabled() {
		searchCategory = v1.SearchCategory_IMAGES_V2
	} else {
		searchCategory = v1.SearchCategory_IMAGES
	}
	return search.NewOptionsMap(searchCategory).Add(search.Cluster, deployments.OptionsMap.MustGet(search.Cluster.String())).
		Add(search.ClusterID, deployments.OptionsMap.MustGet(search.ClusterID.String())).
		Add(search.Namespace, deployments.OptionsMap.MustGet(search.Namespace.String())).
		Add(search.NamespaceID, deployments.OptionsMap.MustGet(search.NamespaceID.String())).
		Add(search.DeploymentLabel, deployments.OptionsMap.MustGet(search.DeploymentLabel.String())).
		Add(search.DeploymentName, deployments.OptionsMap.MustGet(search.DeploymentName.String())).
		Add(search.DeploymentID, deployments.OptionsMap.MustGet(search.DeploymentID.String()))
}()
