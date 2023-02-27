package cvehelper

import (
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// ID creates a CVE ID from the given cve id (and os if postgres is enabled).
func ID(cve, os string) string {
	return pgSearch.IDFromPks([]string{cve, os})
}
