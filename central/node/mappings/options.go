package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// VulnerabilityOptionsMap defines the search options for Vulnerabilities stored in nodes.
var VulnerabilityOptionsMap = search.Walk(v1.SearchCategory_VULNERABILITIES, "node.scan.components.vulns", (*storage.EmbeddedVulnerability)(nil))

// ComponentOptionsMap defines the search options for node components stored in nodes.
// Note: node components are the same as image components for search.
var ComponentOptionsMap = search.Walk(v1.SearchCategory_IMAGE_COMPONENTS, "node.scan.components", (*storage.EmbeddedNodeScanComponent)(nil))
