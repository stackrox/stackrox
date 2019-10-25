package mappings

import (
	"github.com/stackrox/rox/central/deployment/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test
var OptionsMap = search.Walk(v1.SearchCategory_IMAGES, "image", (*storage.Image)(nil)).
	Add(search.Cluster, mappings.OptionsMap.MustGet(search.Cluster.String())).
	Add(search.ClusterID, mappings.OptionsMap.MustGet(search.ClusterID.String())).
	Add(search.Namespace, mappings.OptionsMap.MustGet(search.Namespace.String())).
	Add(search.Label, mappings.OptionsMap.MustGet(search.Label.String())).
	Add(search.DeploymentName, mappings.OptionsMap.MustGet(search.DeploymentName.String())).
	Add(search.DeploymentID, mappings.OptionsMap.MustGet(search.DeploymentID.String()))

// VulnerabilityOptionsMap defines the search options for Vulnerabilities stored in images.
var VulnerabilityOptionsMap = search.Walk(v1.SearchCategory_VULNERABILITIES, "image.scan.components.vulns", (*storage.EmbeddedVulnerability)(nil))

// ComponentOptionsMap defines the search options for image components stored in images.
var ComponentOptionsMap = search.Walk(v1.SearchCategory_IMAGE_COMPONENTS, "image.scan.components", (*storage.EmbeddedImageScanComponent)(nil))
