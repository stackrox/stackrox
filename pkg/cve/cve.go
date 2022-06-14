package cve

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/search/postgres"
)

var (
	log = logging.LoggerForModule()
)

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

// ID creates a CVE ID from the given cve id (and os if postgres is enabled).
func ID(cve, os string) string {
	if features.PostgresDatastore.Enabled() {
		return postgres.IDFromPks([]string{cve, os})
	}
	return cve
}
