package cve

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/utils"
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

// IDV2 creates a CVE ID from the given cve name and component id.
func IDV2(cve, componentID string) string {
	return pgSearch.IDFromPks([]string{cve, componentID})
}

// IDToParts return the CVE ID parts—cve and operating system.
func IDToParts(id string) (string, string) {
	parts := pgSearch.IDToParts(id)
	if len(parts) > 2 {
		utils.Should(errors.Errorf("unexpected number of parts for CVE ID %s", id))
		return "", ""
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}
