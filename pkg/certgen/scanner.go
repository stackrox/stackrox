package certgen

import (
	"github.com/stackrox/rox/pkg/mtls"
)

// IssueScannerCerts issues a cert for the scanner and scanner DB, given a fileMap that contains a ca-cert and ca-key.
// The issued cert and key are added to the passed in fileMap.
// It is extracted out to avoid duplicating the generating code and the file names between central and roxctl,
// and is not intended to be more generally reusable.
func IssueScannerCerts(fileMap map[string][]byte, ca mtls.CA, opts ...mtls.IssueCertOption) error {
	return IssueOtherServiceCerts(fileMap, ca, []mtls.Subject{mtls.ScannerSubject, mtls.ScannerDBSubject}, opts...)
}
