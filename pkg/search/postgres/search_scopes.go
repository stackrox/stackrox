package postgres

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
)

var (
	nodeSearchScope = map[v1.SearchCategory]struct{}{
		v1.SearchCategory_NODE_VULNERABILITIES:    {},
		v1.SearchCategory_NODE_COMPONENT_CVE_EDGE: {},
		v1.SearchCategory_NODE_COMPONENTS:         {},
		v1.SearchCategory_NODE_COMPONENT_EDGE:     {},
		v1.SearchCategory_NODES:                   {},
		v1.SearchCategory_CLUSTERS:                {},
	}

	imageSearchScope = map[v1.SearchCategory]struct{}{
		v1.SearchCategory_IMAGE_VULNERABILITIES: {},
		v1.SearchCategory_COMPONENT_VULN_EDGE:   {},
		v1.SearchCategory_IMAGE_COMPONENTS:      {},
		v1.SearchCategory_IMAGE_COMPONENT_EDGE:  {},
		v1.SearchCategory_IMAGE_VULN_EDGE:       {},
		v1.SearchCategory_IMAGES:                {},
		v1.SearchCategory_DEPLOYMENTS:           {},
		v1.SearchCategory_NAMESPACES:            {},
		v1.SearchCategory_CLUSTERS:              {},
	}

	searchScope = map[v1.SearchCategory]map[v1.SearchCategory]struct{}{
		v1.SearchCategory_IMAGE_VULNERABILITIES: imageSearchScope,
		v1.SearchCategory_COMPONENT_VULN_EDGE:   imageSearchScope,
		v1.SearchCategory_IMAGE_COMPONENTS:      imageSearchScope,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE:  imageSearchScope,
		v1.SearchCategory_IMAGE_VULN_EDGE:       imageSearchScope,
		v1.SearchCategory_IMAGES:                imageSearchScope,
		v1.SearchCategory_DEPLOYMENTS:           imageSearchScope,
		v1.SearchCategory_NAMESPACES:            imageSearchScope,

		v1.SearchCategory_NODE_VULNERABILITIES:    nodeSearchScope,
		v1.SearchCategory_NODE_COMPONENT_CVE_EDGE: nodeSearchScope,
		v1.SearchCategory_NODE_COMPONENTS:         nodeSearchScope,
		v1.SearchCategory_NODE_COMPONENT_EDGE:     nodeSearchScope,
		v1.SearchCategory_NODES:                   nodeSearchScope,

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
