package certgen

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/mtls"
)

func issueCert(fileMap map[string][]byte, subject mtls.Subject, fileNamePrefix string) error {
	cert, err := mtls.IssueNewCertFromCA(subject, fileMap["ca.pem"], fileMap["ca-key.pem"])
	if err != nil {
		return errors.Wrapf(err, "could not issue cert for %s", subject.Identifier)
	}
	fileMap[fmt.Sprintf("%scert.pem", fileNamePrefix)] = cert.CertPEM
	fileMap[fmt.Sprintf("%skey.pem", fileNamePrefix)] = cert.KeyPEM
	return nil

}

// IssueCentralCert issues a central cert, given a fileMap that contains a ca-cert and ca-key.
// The issued cert and key are added to the passed in fileMap.
// It is extracted out to avoid duplicating the generating code and the file names between central and roxctl,
// and is not intended to be more generally reusable.
func IssueCentralCert(fileMap map[string][]byte) error {
	if err := issueCert(fileMap, mtls.CentralSubject, ""); err != nil {
		return err
	}
	return nil
}

// IssueScannerCerts issues a cert for the scanner and scanner DB, given a fileMap that contains a ca-cert and ca-key.
// The issued cert and key are added to the passed in fileMap.
// It is extracted out to avoid duplicating the generating code and the file names between central and roxctl,
// and is not intended to be more generally reusable.
func IssueScannerCerts(fileMap map[string][]byte) error {
	if err := issueCert(fileMap, mtls.ScannerSubject, "scanner-"); err != nil {
		return err
	}

	if err := issueCert(fileMap, mtls.ScannerDBSubject, "scanner-db-"); err != nil {
		return err
	}
	return nil
}
