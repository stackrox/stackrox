package cve

import "github.com/stackrox/rox/generated/storage"

var clusterCVETypes = map[storage.CVE_CVEType]struct{}{
	storage.CVE_ISTIO_CVE:     {},
	storage.CVE_K8S_CVE:       {},
	storage.CVE_OPENSHIFT_CVE: {},
}

var componentCVETypes = map[storage.CVE_CVEType]struct{}{
	storage.CVE_IMAGE_CVE: {},
	storage.CVE_NODE_CVE:  {},
}

// ContainsCVEType returns whether or not typ exists in the cve type slice
func ContainsCVEType(types []storage.CVE_CVEType, typ storage.CVE_CVEType) bool {
	for _, t := range types {
		if t == typ {
			return true
		}
	}
	return false
}

// ContainsComponentBasedCVE returns true if a component-based CVE type exists in the type slice
func ContainsComponentBasedCVE(types []storage.CVE_CVEType) bool {
	for _, t := range types {
		if _, ok := componentCVETypes[t]; ok {
			return true
		}
	}
	return false
}

// ContainsClusterCVE returns true if a cluster CVE type exists in the type slice
func ContainsClusterCVE(types []storage.CVE_CVEType) bool {
	for _, t := range types {
		if _, ok := clusterCVETypes[t]; ok {
			return true
		}
	}
	return false
}
