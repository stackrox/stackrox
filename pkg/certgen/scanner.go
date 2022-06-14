package certgen

import (
	"github.com/stackrox/stackrox/pkg/mtls"
)

// IssueScannerCerts issues a cert for the scanner and scanner DB, given a fileMap that contains a ca-cert and ca-key.
// The issued cert and key are added to the passed in fileMap.
// It is extracted out to avoid duplicating the generating code and the file names between central and roxctl,
// and is not intended to be more generally reusable.
func IssueScannerCerts(fileMap map[string][]byte, ca mtls.CA) error {
	return IssueOtherServiceCerts(fileMap, ca, mtls.ScannerSubject, mtls.ScannerDBSubject)
}
