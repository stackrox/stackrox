package certgen

import (
	"github.com/stackrox/stackrox/pkg/mtls"
)

// IssueCentralCert issues a central cert, given a fileMap that contains a ca-cert and ca-key.
// The issued cert and key are added to the passed in fileMap.
// It is extracted out to avoid duplicating the generating code and the file names between central and roxctl,
// and is not intended to be more generally reusable.
func IssueCentralCert(fileMap map[string][]byte, ca mtls.CA, opts ...mtls.IssueCertOption) error {
	if err := IssueServiceCert(fileMap, ca, mtls.CentralSubject, "", opts...); err != nil {
		return err
	}
	return nil
}
