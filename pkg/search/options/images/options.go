package images

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
)

var (

	// OptionsMap describes the options for Images
	OptionsMap = search.Walk(v1.SearchCategory_IMAGES, "image", (*storage.Image)(nil))

	// ImageComponentEdgeOptionsMap defines the search options for Vulnerabilities stored in images.
	ImageComponentEdgeOptionsMap = search.Walk(v1.SearchCategory_IMAGE_COMPONENT_EDGE, "imagecomponentedge", (*storage.ImageComponentEdge)(nil))

	// ImageComponentOptionsMap defines the search options for image components stored in images.
	ImageComponentOptionsMap = search.Walk(v1.SearchCategory_IMAGE_COMPONENTS, "image_component", (*storage.ImageComponent)(nil))

	// ImageComponentCVEEdgeOptionsMap defines the search options for Vulnerabilities stored in images.
	ImageComponentCVEEdgeOptionsMap = search.Walk(v1.SearchCategory_COMPONENT_VULN_EDGE, "component_c_v_e_edge", (*storage.ComponentCVEEdge)(nil))

	// ImageCVEOptionsMap defines the search options for Vulnerabilities stored in images.
	ImageCVEOptionsMap = search.Walk(v1.SearchCategory_VULNERABILITIES, "c_v_e", (*storage.CVE)(nil))

	// ImageCVEEdgeOptionsMap defines the search options for vulnerabilities in images.
	ImageCVEEdgeOptionsMap = search.Walk(v1.SearchCategory_IMAGE_VULN_EDGE, "image_c_v_e_edge", (*storage.ImageCVEEdge)(nil))

	// FullImageOptionsMap defined the options for image that includes components and vulns options.
	FullImageOptionsMap = search.CombineOptionsMaps(
		OptionsMap,
		ImageComponentEdgeOptionsMap,
		ImageComponentOptionsMap,
		ImageComponentCVEEdgeOptionsMap,
		ImageCVEOptionsMap,
		ImageCVEEdgeOptionsMap,
	)

	// ImageDeploymentOptions defines the deployment options available to search on images
	ImageDeploymentOptions = search.NewOptionsMap(v1.SearchCategory_IMAGES).Add(search.Cluster, deployments.OptionsMap.MustGet(search.Cluster.String())).
				Add(search.ClusterID, deployments.OptionsMap.MustGet(search.ClusterID.String())).
				Add(search.Namespace, deployments.OptionsMap.MustGet(search.Namespace.String())).
				Add(search.NamespaceID, deployments.OptionsMap.MustGet(search.NamespaceID.String())).
				Add(search.Label, deployments.OptionsMap.MustGet(search.Label.String())).
				Add(search.DeploymentName, deployments.OptionsMap.MustGet(search.DeploymentName.String())).
				Add(search.DeploymentID, deployments.OptionsMap.MustGet(search.DeploymentID.String()))
)
