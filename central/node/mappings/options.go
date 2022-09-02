package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// VulnerabilityOptionsMap defines the search options for Vulnerabilities stored in nodes.
var VulnerabilityOptionsMap = search.Walk(v1.SearchCategory_VULNERABILITIES, "node.scan.components.vulns", (*storage.EmbeddedVulnerability)(nil))

// ComponentOptionsMap defines the search options for node components stored in nodes.
// Note: node components are the same as image components for search.
var ComponentOptionsMap = search.Walk(v1.SearchCategory_IMAGE_COMPONENTS, "node.scan.components", (*storage.EmbeddedNodeScanComponent)(nil))

var NodeComponentOptionsMap = search.Walk(v1.SearchCategory_NODE_COMPONENTS, "node.scan.components", (*storage.EmbeddedNodeScanComponent)(nil))

// NodeVulnerabilityOptionsMap defines the search options for NodeVulnerabilities stores in node scan
var NodeVulnerabilityOptionsMap = search.Walk(v1.SearchCategory_NODE_VULNERABILITIES, "node.scan.components.vulnerabilities", (*storage.NodeVulnerability)(nil))
