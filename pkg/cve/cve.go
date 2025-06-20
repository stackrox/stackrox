package cve

import (
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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

// ID creates a CVE ID from the given cve id and os.
func ID(cve, os string) string {
	return pgSearch.IDFromPks([]string{cve, os})
}

// IDV2 creates a CVE ID from the given cve name, component id and index of CVE within the component.
func IDV2(cve *storage.EmbeddedVulnerability, componentID string) (string, error) {
	hash, err := hashstructure.Hash(cve, hashstructure.FormatV2, &hashstructure.HashOptions{ZeroNil: true})
	if err != nil {
		return "", err
	}

	return pgSearch.IDFromPks([]string{cve.GetCve(), strconv.FormatUint(hash, 10), componentID}), nil
}

// IDToParts return the CVE ID parts—cve and operating system.
func IDToParts(id string) (string, string) {
	parts := pgSearch.IDToParts(id)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}
