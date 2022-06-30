package postgres

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
)

var (
	nodeTree = map[v1.SearchCategory]struct{}{
		v1.SearchCategory_NODE_VULNERABILITIES:    {},
		v1.SearchCategory_NODE_COMPONENT_CVE_EDGE: {},
		v1.SearchCategory_NODE_COMPONENTS:         {},
		v1.SearchCategory_NODE_COMPONENT_EDGE:     {},
		v1.SearchCategory_NODES:                   {},
		v1.SearchCategory_CLUSTERS:                {},
	}

	imageTree = map[v1.SearchCategory]struct{}{
		v1.SearchCategory_IMAGE_VULNERABILITIES: {},
		v1.SearchCategory_COMPONENT_VULN_EDGE:   {},
		v1.SearchCategory_IMAGE_COMPONENTS:      {},
		v1.SearchCategory_IMAGE_COMPONENT_EDGE:  {},
		v1.SearchCategory_IMAGES:                {},
		v1.SearchCategory_DEPLOYMENTS:           {},
		v1.SearchCategory_NAMESPACES:            {},
		v1.SearchCategory_CLUSTERS:              {},
	}

	searchScope = map[v1.SearchCategory]map[v1.SearchCategory]struct{}{
		v1.SearchCategory_IMAGE_VULNERABILITIES: imageTree,
		v1.SearchCategory_IMAGE_COMPONENTS:      imageTree,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE:  imageTree,
		v1.SearchCategory_IMAGES:                imageTree,
		v1.SearchCategory_DEPLOYMENTS:           imageTree,
		v1.SearchCategory_NAMESPACES:            imageTree,

		v1.SearchCategory_NODE_VULNERABILITIES:    nodeTree,
		v1.SearchCategory_NODE_COMPONENT_CVE_EDGE: nodeTree,
		v1.SearchCategory_NODE_COMPONENTS:         nodeTree,
		v1.SearchCategory_NODE_COMPONENT_EDGE:     nodeTree,
		v1.SearchCategory_NODES:                   nodeTree,

		// for testing

		// TestChild1P4
		72: {
			// TestGrandparents
			61: {},
			// TestChild1P4
			74: {},
		},
		// TestParent4
		74: {
			// TestChild1P4
			74: {},
		},
	}
)
