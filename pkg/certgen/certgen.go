package certgen

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/services"
)

// AddCertToFileMap adds `cert.pem` and `key.pem` entries for the given certificate (prefixed with
// fileNamePrefix, if any) to the given file map.
func AddCertToFileMap(fileMap map[string][]byte, cert *mtls.IssuedCert, fileNamePrefix string) {
	fileMap[fileNamePrefix+mtls.ServiceCertFileName] = cert.CertPEM
	fileMap[fileNamePrefix+mtls.ServiceKeyFileName] = cert.KeyPEM
}

// IssueServiceCert issues a certificate for the given service, and stores it in the given fileMap
// (keys prefixed with fileNamePrefix, if any).
func IssueServiceCert(fileMap map[string][]byte, ca mtls.CA, subject mtls.Subject, fileNamePrefix string, opts ...mtls.IssueCertOption) error {
	cert, err := ca.IssueCertForSubject(subject, opts...)
	if err != nil {
		return errors.Wrapf(err, "could not issue cert for %s", subject.Identifier)
	}
	AddCertToFileMap(fileMap, cert, fileNamePrefix)
	return nil
}

// IssueOtherServiceCerts issues certificates for the given subjects, and stores them in the given file
// map. The file name prefix is chosen as the slug-case of the service type plus a trailing hyphen.
func IssueOtherServiceCerts(fileMap map[string][]byte, ca mtls.CA, subjs []mtls.Subject, opts ...mtls.IssueCertOption) error {
	for _, subj := range subjs {
		if err := IssueServiceCert(fileMap, ca, subj, services.ServiceTypeToSlugName(subj.ServiceType)+"-", opts...); err != nil {
			return err
		}
	}
	return nil
}

// VerifyServiceCertAndKey verifies that the service certificate (stored with the given fileNamePrefix in the file
// map) is a valid service certificate for the given serviceType, relative to the given CA.
// It also verifies that the associated private key in the file map also matches the certificate.
// If currentTime is non-nil, it is used as the reference time for certificate validity; otherwise, the current system time is used.
func VerifyServiceCertAndKey(fileMap map[string][]byte, fileNamePrefix string, ca mtls.CA, serviceType storage.ServiceType,
	currentTime *time.Time, extraValidations ...func(certificate *x509.Certificate) error) error {
	certPEM := fileMap[fileNamePrefix+mtls.ServiceCertFileName]
	if len(certPEM) == 0 {
		return fmt.Errorf("no service certificate for %s in file map", serviceType.String())
	}
	cert, err := helpers.ParseCertificatePEM(certPEM)
	if err != nil {
		return errors.New("unparseable certificate in file map")
	}

	var verifyOpts []mtls.VerifyCertOption
	if currentTime != nil {
		verifyOpts = append(verifyOpts, mtls.WithCurrentTime(*currentTime))
	}
	subjFromCert, err := ca.ValidateAndExtractSubject(cert, verifyOpts...)
	if err != nil {
		return errors.Wrap(err, "failed to validate certificate and extract subject")
	}
	if subjFromCert.ServiceType != serviceType {
		return errors.Errorf("unexpected certificate service type: got %s, expected %s", subjFromCert.ServiceType, serviceType)
	}

	for _, validation := range extraValidations {
		if err := validation(cert); err != nil {
			return errors.Wrap(err, "failed to validate certificate")
		}
	}

	keyPEM := fileMap[fileNamePrefix+mtls.ServiceKeyFileName]
	if len(keyPEM) == 0 {
		return fmt.Errorf("no service private key for %s in file map", serviceType.String())
	}

	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return errors.Wrap(err, "mismatched certificate and private key, or invalid private key")
	}

	return nil
}
