package images

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
)

// OptionsMap describes the options for Images
var OptionsMap = search.Walk(v1.SearchCategory_IMAGES, "image", (*storage.Image)(nil))

// ImageDeploymentOptions defines the deployment options available to search on images
var ImageDeploymentOptions = search.NewOptionsMap(v1.SearchCategory_IMAGES).Add(search.Cluster, deployments.OptionsMap.MustGet(search.Cluster.String())).
	Add(search.ClusterID, deployments.OptionsMap.MustGet(search.ClusterID.String())).
	Add(search.Namespace, deployments.OptionsMap.MustGet(search.Namespace.String())).
	Add(search.NamespaceID, deployments.OptionsMap.MustGet(search.NamespaceID.String())).
	Add(search.DeploymentLabel, deployments.OptionsMap.MustGet(search.DeploymentLabel.String())).
	Add(search.DeploymentName, deployments.OptionsMap.MustGet(search.DeploymentName.String())).
	Add(search.DeploymentID, deployments.OptionsMap.MustGet(search.DeploymentID.String()))
