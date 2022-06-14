package images

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/options/deployments"
)

// OptionsMap describes the options for Images
var OptionsMap = search.Walk(v1.SearchCategory_IMAGES, "image", (*storage.Image)(nil))

// ImageDeploymentOptions defines the deployment options available to search on images
var ImageDeploymentOptions = search.NewOptionsMap(v1.SearchCategory_IMAGES).Add(search.Cluster, deployments.OptionsMap.MustGet(search.Cluster.String())).
	Add(search.ClusterID, deployments.OptionsMap.MustGet(search.ClusterID.String())).
	Add(search.Namespace, deployments.OptionsMap.MustGet(search.Namespace.String())).
	Add(search.NamespaceID, deployments.OptionsMap.MustGet(search.NamespaceID.String())).
	Add(search.Label, deployments.OptionsMap.MustGet(search.Label.String())).
	Add(search.DeploymentName, deployments.OptionsMap.MustGet(search.DeploymentName.String())).
	Add(search.DeploymentID, deployments.OptionsMap.MustGet(search.DeploymentID.String()))
