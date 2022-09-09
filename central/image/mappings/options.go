package mappings

import (
	"github.com/stackrox/rox/pkg/search/options/images"
)

// OptionsMap defines the search options for Image.
var OptionsMap = images.FullImageOptionsMap

// VulnerabilityOptionsMap defines the search options for Vulnerabilities stored in images.
var VulnerabilityOptionsMap = images.ImageCVEOptionsMap

// ComponentOptionsMap defines the search options for image components stored in images.
var ComponentOptionsMap = images.ImageComponentOptionsMap
