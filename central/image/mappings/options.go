package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// VulnerabilityOptionsMap defines the search options for Vulnerabilities stored in images.
var VulnerabilityOptionsMap = search.Walk(v1.SearchCategory_VULNERABILITIES, "image.scan.components.vulns", (*storage.EmbeddedVulnerability)(nil))

// ComponentOptionsMap defines the search options for image components stored in images.
var ComponentOptionsMap = search.Walk(v1.SearchCategory_IMAGE_COMPONENTS, "image.scan.components", (*storage.EmbeddedImageScanComponent)(nil))
