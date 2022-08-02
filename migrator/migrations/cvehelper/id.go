package cvehelper

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/utils"
)

// ID creates a CVE ID from the given cve id (and os if postgres is enabled).
func ID(cve, os string) string {
	return postgres.IDFromPks([]string{cve, os})
}

// IDToParts return the CVE ID parts—cve and operating system.
func IDToParts(id string) (string, string) {
	parts := postgres.IDToParts(id)
	if len(parts) > 2 {
		utils.Should(errors.Errorf("unexpected number of parts for CVE ID %s", id))
		return "", ""
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}
